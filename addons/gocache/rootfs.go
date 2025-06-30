package gocache

import (
	"os"
	"path/filepath"
)

type rootFS struct {
	root *os.Root
	diskDirBaseFS
}

func wrapRoot(root *os.Root) diskDirFS {
	return &rootFS{root, root.FS().(diskDirBaseFS)}
}

func (r *rootFS) Name() string {
	return r.root.Name()
}

func (r *rootFS) OpenFile(name string, flag int, perm os.FileMode) (writeFile, error) {
	return r.root.OpenFile(name, flag, perm)
}

func (r *rootFS) FullName(name string) string {
	return filepath.Join(r.root.Name(), name)
}

func (r *rootFS) Rename(oldpath, newpath string) error {
	return os.Rename(r.FullName(oldpath), r.FullName(newpath))
}

func (r *rootFS) Remove(name string) error {
	return r.root.Remove(name)
}
