package gocache

type RemoteStorageFactory interface {
	Want(url string) bool
	New(url string) (ReadonlyStorageBackend, error)
}
