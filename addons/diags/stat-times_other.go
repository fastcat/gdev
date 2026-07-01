//go:build !linux

package diags

import (
	"archive/tar"
	"io/fs"
)

func fillPlatformStatFields(_ *tar.Header, _ fs.FileInfo) {
	// On non-Linux platforms, Uid, Gid, AccessTime, and ChangeTime
	// are left at their zero values. Darwin uses different syscall.Stat_t
	// field names (Atimespec/Ctimespec) and Windows has no syscall.Stat_t.
}
