package diags

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sync"
	"syscall"
	"time"

	"fastcat.org/go/gdev/instance"
)

type fileOpener func(context.Context) (*os.File, io.WriteCloser, error)

type TarFileCollector struct {
	// Opener is a required function that opens the underlying destination file.
	//
	// It returns the destination as two pieces, one as the actual underlying
	// file, and the other as the writer to write to, which may be a different
	// object such as a gzip compressor.
	Opener fileOpener
	mu     sync.Mutex
	dest   *os.File
	out    io.WriteCloser
	tw     *tar.Writer
	// set true if we get an error writing to the tar that indicates we can't
	// usefully write any more entries to it.
	twFatal bool
}

// Begin implements Collector.
func (f *TarFileCollector) Begin(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.dest != nil {
		return fmt.Errorf("collector already begun")
	}
	var err error
	f.dest, f.out, err = f.Opener(ctx)
	if err != nil {
		if f.out != nil && f.out != f.dest {
			_ = f.out.Close()
		}
		if f.dest != nil {
			_ = f.dest.Close()
			_ = os.Remove(f.dest.Name())
		}
		return err
	}
	f.tw = tar.NewWriter(f.out)
	return nil
}

func fillTarHeader(th *tar.Header, contents io.Reader) (bool, error) {
	// Stat() covers os.File and fs.File
	// NOTE: os.FileInfo === fs.FileInfo
	if s, ok := contents.(interface{ Stat() (fs.FileInfo, error) }); ok {
		// os.File, fs.File, and similar
		fi, err := s.Stat()
		if err != nil {
			return false, fmt.Errorf("error getting file info: %w", err)
		}
		th.Size = fi.Size()
		th.ModTime = fi.ModTime()
		th.Mode = int64(fi.Mode().Perm())
		if fis, ok := fi.Sys().(syscall.Stat_t); ok {
			// don't bother trying to look up user information
			th.Uid, th.Gid = int(fis.Uid), int(fis.Gid)
			th.AccessTime = time.Unix(fis.Atim.Unix())
			th.ChangeTime = time.Unix(fis.Ctim.Unix())
		}
		return true, nil
	}

	// Seeker covers bytes.Reader and likely others
	if s, ok := contents.(io.Seeker); ok {
		// covers bytes.Reader
		// get size by seeking to the end and back
		if pos, err := s.Seek(0, io.SeekCurrent); err != nil {
			return false, fmt.Errorf("error seeking to get size: %w", err)
		} else if th.Size, err = s.Seek(0, io.SeekEnd); err != nil {
			return false, fmt.Errorf("error seeking to end: %w", err)
		} else if _, err := s.Seek(pos, io.SeekStart); err != nil {
			return false, fmt.Errorf("error seeking back to original position: %w", err)
		}
		return true, nil
	}

	if s, ok := contents.(interface{ Len() int }); ok {
		// covers bytes.Buffer and similar
		th.Size = int64(s.Len())
		return true, nil
	}

	return false, nil
}

// Collect implements Collector.
func (f *TarFileCollector) Collect(ctx context.Context, name string, contents io.Reader) error {
	th := &tar.Header{
		Name:     name,
		Mode:     0o644,
		Typeflag: tar.TypeReg,
		ModTime:  time.Now(),
	}
	if ok, err := fillTarHeader(th, contents); err != nil {
		return err
	} else if !ok {
		// TODO: support bytes.Buffer via Len()
		// stream to a temp file and then use that for the collection
		tf, err := os.CreateTemp("", instance.AppName()+"-diags-coll-*")
		if err != nil {
			return fmt.Errorf("error creating temp file for contents for %s: %w", name, err)
		}
		// pre-delete the file, avoids more complex error handling later, and
		// reduces secrets exposure. this only works on unix-y platforms.
		if err := os.Remove(tf.Name()); err != nil {
			_ = tf.Close()
			return fmt.Errorf("error removing temp file %s for collecting %s: %w", tf.Name(), name, err)
		}
		defer tf.Close() //nolint:errcheck
		if _, err := io.Copy(tf, contents); err != nil {
			return fmt.Errorf("cannot determine size of contents, not seekable")
		} else if _, err := tf.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("error seeking to start of temp file %s for collecting %s: %w", tf.Name(), name, err)
		}
		if ok, err := fillTarHeader(th, tf); err != nil {
			return fmt.Errorf("error filling tar header for %s: %w", name, err)
		} else if !ok {
			// TODO: bug detection, should we panic?
			return fmt.Errorf("probable bug: cannot get tar header info for temp file %s for collecting %s", tf.Name(), name)
		}
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if f.dest == nil || f.out == nil {
		return fmt.Errorf("collector not begun")
	}

	if err := f.tw.WriteHeader(th); err != nil {
		f.twFatal = true
		return fmt.Errorf("error writing tar header for %s: %w", name, err)
	} else if _, err := io.Copy(f.tw, contents); err != nil {
		f.twFatal = true
		return fmt.Errorf("error writing contents for %s to tar: %w", name, err)
	}

	return nil
}

// Destination implements Collector.
func (f *TarFileCollector) Destination() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.dest == nil {
		return "(not started)"
	}
	return f.dest.Name()
}

// Finalize implements Collector.
func (f *TarFileCollector) Finalize(ctx context.Context, collectErr error) error {
	// TODO: try to collect the collectErr itself into the file

	f.mu.Lock()
	defer f.mu.Unlock()

	// if anything in the flush/close chain fails, we keep going because we always
	// want to get to actually closing the real file, and we just join up any
	// errors we hit along the way.
	var errs []error

	if err := f.tw.Close(); err != nil {
		errs = append(errs, err)
	}
	if f.out != f.dest {
		if err := f.out.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if err := f.dest.Sync(); err != nil {
		errs = append(errs, err)
	}
	if err := f.dest.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) == 0 {
		return nil
	} else if len(errs) == 1 {
		return errs[0]
	} else {
		return errors.Join(errs...)
	}
}

var _ Collector = (*TarFileCollector)(nil)

// OpenTempDiagsFile creates a new temporary file in the system temp directory,
// named with the app name, and compressed with default gzip compression.
//
// The resulting filename will usually be along the lines of `/tmp/xdev-diags-XXX.tgz`,
// where `XXX` is a random string of indeterminate length.
func OpenTempDiagsFile(context.Context) (*os.File, io.WriteCloser, error) {
	// TODO: add a timestamp to the filename?
	f, err := os.CreateTemp(os.TempDir(), instance.AppName()+"-diags-*.tgz")
	if err != nil {
		return nil, nil, err
	}
	w := gzip.NewWriter(f)
	return f, w, nil
}
