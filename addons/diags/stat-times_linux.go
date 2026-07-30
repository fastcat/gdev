package diags

import (
	"archive/tar"
	"io/fs"
	"syscall"
	"time"
)

func fillPlatformStatFields(th *tar.Header, fi fs.FileInfo) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return
	}
	th.Uid, th.Gid = int(st.Uid), int(st.Gid)
	th.AccessTime = time.Unix(st.Atim.Unix())
	th.ChangeTime = time.Unix(st.Ctim.Unix())
}
