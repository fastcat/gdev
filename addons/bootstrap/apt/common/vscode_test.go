package apt_common

import "testing"

func TestVSCodeArchiveKeyring(t *testing.T) {
	// this will panic on errors, we want to ensure it does not panic
	_ = VSCodeArchiveKeyringBinary()
}
