package textedit

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"hash/crc32"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"fastcat.org/go/gdev/lib/sys"
)

func EditFile(
	fileName string,
	editor Editor,
) (bool, error) {
	var in io.ReadCloser
	var err error
	in, err = os.Open(fileName)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		// file doesn't exist, so we can just create it as if there was an empty file there
		in = io.NopCloser(bytes.NewReader(nil))
	}
	defer in.Close() // nolint:errcheck
	d := filepath.Dir(fileName)
	out, err := os.CreateTemp(d, filepath.Base(fileName)+".tmp")
	if err != nil {
		return false, err
	}
	defer out.Close() // nolint:errcheck
	// keep a running checksum so we know if we can skip the final rename due to not
	// making any changes. This doesn't need to be a strong hash.
	hIn, hOut := crc32.NewIEEE(), crc32.NewIEEE()
	mr := io.TeeReader(in, hIn)
	mw := io.MultiWriter(hOut, out)
	if err := Edit(mr, mw, editor); err != nil {
		_ = out.Close()
		_ = os.Remove(out.Name())
		return false, err
	}
	// protect user data: flush the new file to disk before we do the rename
	if err := out.Sync(); err != nil {
		_ = out.Close()
		_ = os.Remove(out.Name())
		return false, err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(out.Name())
		return false, err
	}
	if err := in.Close(); err != nil {
		return false, err
	}
	// if the checksums match, we didn't make any changes, so we can skip the
	// rename and avoid the mtime/etc update of the file.
	if hIn.Sum32() == hOut.Sum32() {
		// we didn't make any changes, so just remove the temp file
		err := os.Remove(out.Name())
		return false, err
	}
	if err := os.Rename(out.Name(), fileName); err != nil {
		_ = os.Remove(out.Name())
		return false, err
	}
	return true, nil
}

func EditFileAsRoot(
	ctx context.Context,
	fileName string,
	editor Editor,
) (bool, error) {
	var in io.ReadCloser
	var err error
	in, err = sys.SudoReaderIfNecessary(ctx, fileName, true)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		// file doesn't exist, so we can just create it as if there was an empty file there
		in = io.NopCloser(bytes.NewReader(nil))
	}
	defer in.Close() // nolint:errcheck
	// create the output in memory so we can do the equality check before the
	// "temp file as root" dance gets messy
	out := bytes.NewBuffer(nil)
	// keep a running checksum so we know if we can skip the final rename due to not
	// making any changes. This doesn't need to be a strong hash.
	hIn, hOut := crc32.NewIEEE(), crc32.NewIEEE()
	mr := io.TeeReader(in, hIn)
	mw := io.MultiWriter(hOut, out)
	if err := Edit(mr, mw, editor); err != nil {
		return false, err
	}
	if err := in.Close(); err != nil {
		return false, err
	}
	// if the checksums match, we didn't make any changes, so we can skip the
	// rename and avoid the mtime/etc update of the file.
	if hIn.Sum32() == hOut.Sum32() {
		// we didn't make any changes
		return false, nil
	}
	// create a new temp file as root in the target dir and write the modified
	// content to it. Use more entropy than os.CreateTemp does because we can't
	// use O_EXCL here.
	tmpFn := fileName + ".tmp-" + strconv.FormatUint(rand.Uint64(), 36)
	if err := sys.WriteFileAsRoot(ctx, tmpFn, bytes.NewReader(out.Bytes()), 0o600); err != nil {
		if err2 := sys.RemoveFileAsRoot(ctx, tmpFn); err2 != nil {
			err = errors.Join(err, err2)
		}
		return false, err
	}
	if err := sys.RenameFileAsRoot(ctx, tmpFn, fileName); err != nil {
		if err2 := sys.RemoveFileAsRoot(ctx, tmpFn); err2 != nil {
			err = errors.Join(err, err2)
		}
		return false, err
	}
	return true, nil
}

func Edit(
	in io.Reader,
	out io.Writer,
	editor Editor,
) error {
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := scanner.Text()
		output, err := editor.Next(line)
		if err != nil {
			return err
		}
		for outLine := range output {
			if !strings.HasSuffix(outLine, "\n") {
				outLine += "\n"
			}
			if _, err := out.Write([]byte(outLine)); err != nil {
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	output, err := editor.EOF()
	if err != nil {
		return err
	}
	for outLine := range output {
		if !strings.HasSuffix(outLine, "\n") {
			outLine += "\n"
		}
		if _, err := out.Write([]byte(outLine)); err != nil {
			return err
		}
	}
	return nil
}
