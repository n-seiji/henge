//go:build !unix

package state

// flock is unavailable here; run one henge per data directory.
func LockDir(string) (func(), error) {
	return func() {}, nil
}
