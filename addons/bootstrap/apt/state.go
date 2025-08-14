package apt

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/shx"
)

type timestampedMap struct {
	timestamp time.Time
	data      map[string]string
}

var (
	installedKey = bootstrap.NewKey[timestampedMap]("dpkg-installed")
	availableKey = bootstrap.NewKey[timestampedMap]("apt-available")
)

const (
	// We can look at the timestamp of the dpkg status file to determine
	// when the package installation state has changed.
	dpkgStatusFile = "/var/lib/dpkg/status"
	// We can look at the timestamp of the apt package cache file to determine
	// when the available packages have changed.
	aptPkgCacheFile = "/var/cache/apt/pkgcache.bin"
)

func DpkgInstalled(ctx *bootstrap.Context) (map[string]string, error) {
	data, ok := bootstrap.Get(ctx, installedKey)
	if ok {
		if st, err := os.Stat(dpkgStatusFile); err != nil ||
			st.ModTime().After(data.timestamp) {
			// on-disk file is newer than in memory cache, invalidate it
			ok = false
			bootstrap.Clear(ctx, installedKey)
		}
	}
	if ok {
		return data.data, nil
	}
	data = timestampedMap{
		timestamp: time.Now(),
		data:      make(map[string]string),
	}
	// get status in lines of tab-separated fields
	// TODO: stream this instead of letting shx buffer it
	res, err := shx.Run(
		ctx,
		[]string{
			"dpkg-query",
			"--show",
			"--showformat",
			"${Package}\t${Version}\t${db:Status-Want}\t${db:Status-Status}\t${db:Status-Eflag}\n",
		},
		shx.CaptureOutput(),
		shx.WithCombinedError(),
	)
	if err != nil {
		return nil, err
	}
	defer res.Close() //nolint:errcheck

	b := bufio.NewReader(res.Stdout())
	for {
		line, err := b.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		fields := strings.Split(strings.TrimSuffix(line, "\n"), "\t")
		if len(fields) != 5 {
			return nil, fmt.Errorf("invalid dpkg-query output: %q", line)
		}
		if fields[2] != "install" || fields[3] != "installed" || fields[4] != "ok" {
			// not installed or in a bad state
			continue
		}
		data.data[fields[0]] = fields[1] // package name -> version
	}
	bootstrap.Save(ctx, installedKey, data)
	return data.data, nil
}

func AptAvailable(ctx *bootstrap.Context) (map[string]string, error) {
	data, ok := bootstrap.Get(ctx, availableKey)
	if ok {
		if st, err := os.Stat(aptPkgCacheFile); err != nil ||
			st.ModTime().After(data.timestamp) {
			// on-disk file is newer than in memory cache, invalidate it
			ok = false
			bootstrap.Clear(ctx, availableKey)
		}
	}
	if ok {
		return data.data, nil
	}
	data = timestampedMap{
		timestamp: time.Now(),
		data:      make(map[string]string),
	}
	// This prints out deb822 style stanzas with blank lines between
	// TODO: stream this instead of letting shx buffer it
	res, err := shx.Run(
		ctx,
		[]string{"apt-cache", "dumpavail"},
		shx.CaptureOutput(),
		shx.WithCombinedError(),
	)
	if err != nil {
		return nil, err
	}
	defer res.Close() //nolint:errcheck

	s := bufio.NewScanner(res.Stdout())
	s.Split(Deb822SplitStanza)
	for s.Scan() {
		stanza := s.Bytes()
		parsed, err := ParseDeb822Stanza(bytes.NewReader(stanza))
		if err != nil {
			return nil, fmt.Errorf("failed to parse apt-cache dumpavail stanza: %w", err)
		}
		pkg, ok := parsed["Package"]
		if !ok {
			return nil, fmt.Errorf("missing Package field in apt-cache dumpavail stanza")
		}
		ver, ok := parsed["Version"]
		if !ok {
			return nil, fmt.Errorf("missing Version field for package %s in apt-cache dumpavail stanza", pkg)
		}
		data.data[pkg] = ver
	}
	bootstrap.Save(ctx, availableKey, data)
	return data.data, nil
}
