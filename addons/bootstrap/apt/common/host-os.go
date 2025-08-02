package apt_common

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

func DpkgHostArchitecture() string {
	arch := runtime.GOARCH
	switch arch {
	// TODO: we don't really want to try to support 32 bit architectures
	case "arm":
		return "armhf"
	case "386":
		return "i386"
	case "wasm":
		panic("WASM architecture is not supported by dpkg")
	default:
		// most architectures match, e.g. amd64 and arm64
		// TODO: this isn't correct for several less common architectures
		// but we don't support those anyway
		return arch
	}
}

var HostOSRelease = sync.OnceValues(func() (*OSRelease, error) {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return nil, fmt.Errorf("failed to open /etc/os-release: %w", err)
	}
	defer file.Close() //nolint:errcheck
	osRelease, err := ParseOSRelease(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse /etc/os-release: %w", err)
	}
	return osRelease, nil
})

func HostOSVersionCodename() string {
	if r, err := HostOSRelease(); err != nil {
		panic(err)
	} else if codename, ok := r.Extra["VERSION_CODENAME"]; ok {
		return codename
	} else {
		panic(fmt.Errorf("no VERSION_CODENAME found in /etc/os-release: %v", r.Extra))
	}
}

type OSRelease struct {
	Name      string
	ID        string
	VersionID string
	Extra     map[string]string
}

func ParseOSRelease(in io.Reader) (*OSRelease, error) {
	// read lines and parse fields
	var out OSRelease
	scanner := bufio.NewScanner(in)
	out.Extra = make(map[string]string)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || line[0] == '#' {
			// skip empty lines and comments
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("invalid line in os-release: %q", line)
		} else if value != "" && value[0] == '"' {
			var err error
			value, err = strconv.Unquote(value)
			if err != nil {
				return nil, fmt.Errorf("invalid quoted value in os-release: %q: %w", value, err)
			}
		}
		switch key {
		case "NAME":
			out.Name = value
		case "ID":
			out.ID = value
		case "VERSION_ID":
			out.VersionID = value
		default:
			out.Extra[key] = value
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return &out, nil
}
