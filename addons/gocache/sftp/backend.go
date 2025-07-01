package gocache_sftp

import (
	"errors"
	"io/fs"
	"net/url"
	"os"
	"path"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"fastcat.org/go/gdev/addons/gocache"
)

type sftpStorageBackend struct {
	sshC    *ssh.Client
	sftpC   *sftp.Client
	baseURL *url.URL
}

// Close implements gocache.DiskDirFS.
func (s *sftpStorageBackend) Close() error {
	var errs []error
	if s.sftpC != nil {
		errs = append(errs, s.sftpC.Close())
		s.sftpC = nil
	}
	if s.sshC != nil {
		errs = append(errs, s.sshC.Close())
		s.sshC = nil
	}
	return errors.Join(errs...)
}

// FullName implements gocache.DiskDirFS.
func (s *sftpStorageBackend) FullName(name string) string {
	u2 := *s.baseURL
	u2.Path = path.Join(s.baseURL.Path, name)
	return u2.String()
}

// Mkdir implements gocache.DiskDirFS.
func (s *sftpStorageBackend) Mkdir(name string, _ fs.FileMode) error {
	p := path.Join(s.baseURL.Path, name)
	// ErrExists doesn't work right over sftp, in addition to the client lib not
	// translating the ErrExists, the sftp server may just return a generic error
	// anyways
	err := s.sftpC.Mkdir(p)
	if err == nil {
		return nil
	}
	if st, err2 := s.sftpC.Stat(p); err2 == nil && st.IsDir() {
		// directory already exists, so we can ignore the error
		return os.ErrExist
	}
	return err
}

// Name implements gocache.DiskDirFS.
func (s *sftpStorageBackend) Name() string {
	return s.baseURL.String()
}

// Open implements gocache.DiskDirFS.
func (s *sftpStorageBackend) Open(name string) (fs.File, error) {
	p := path.Join(s.baseURL.Path, name)
	f, err := s.sftpC.Open(p)
	return f, err
}

// OpenFile implements gocache.DiskDirFS.
func (s *sftpStorageBackend) OpenFile(name string, flag int, _ fs.FileMode) (gocache.WriteFile, error) {
	p := path.Join(s.baseURL.Path, name)
	f, err := s.sftpC.OpenFile(p, flag)
	return f, err
}

// Remove implements gocache.DiskDirFS.
func (s *sftpStorageBackend) Remove(name string) error {
	p := path.Join(s.baseURL.Path, name)
	err := s.sftpC.Remove(p)
	return err
}

// Rename implements gocache.DiskDirFS.
func (s *sftpStorageBackend) Rename(oldpath, newpath string) error {
	old := path.Join(s.baseURL.Path, oldpath)
	new := path.Join(s.baseURL.Path, newpath)
	// sftp rename may fail if the target exists
	_ = s.sftpC.Remove(new)
	err := s.sftpC.Rename(old, new)
	return err
}

// Stat implements gocache.DiskDirFS.
func (s *sftpStorageBackend) Stat(name string) (fs.FileInfo, error) {
	p := path.Join(s.baseURL.Path, name)
	st, err := s.sftpC.Stat(p)
	return st, err
}
