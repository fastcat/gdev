package gocache

import (
	"io/fs"
	"os"
	"path/filepath"
)

type rootFS struct {
	os.Root
}

func (r *rootFS) FullName(name string) string {
	return filepath.Join(r.Root.Name(), name)
}

// Open implements DiskDirFS.
// Subtle: this method shadows the method (Root).Open of rootFS.Root to change the return type.
func (r *rootFS) Open(name string) (fs.File, error) {
	return r.Root.Open(name)
}

// OpenFile implements DiskDirFS.
// Subtle: this method shadows the method (Root).OpenFile of rootFS.Root to change the return type.
func (r *rootFS) OpenFile(name string, flag int, perm fs.FileMode) (WriteFile, error) {
	return r.Root.OpenFile(name, flag, perm)
}
