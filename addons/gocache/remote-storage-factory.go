package gocache

type RemoteStorageFactory interface {
	Want(uri string) bool
	New(uri string) (ReadonlyStorageBackend, error)
}
