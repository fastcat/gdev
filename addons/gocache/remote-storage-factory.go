package gocache

type RemoteStorageFactory interface {
	Name() string
	Want(uri string) bool
	New(uri string) (ReadonlyStorageBackend, error)
}
