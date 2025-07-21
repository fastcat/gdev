package textedit

import (
	"bufio"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func EditFile(
	fileName string,
	editor Editor,
) error {
	in, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer in.Close() // nolint:errcheck
	d := filepath.Dir(fileName)
	out, err := os.CreateTemp(d, filepath.Base(fileName)+".tmp")
	if err != nil {
		return err
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
		return err
	}
	// protect user data: flush the new file to disk before we do the rename
	if err := out.Sync(); err != nil {
		_ = out.Close()
		_ = os.Remove(out.Name())
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(out.Name())
		return err
	}
	if err := in.Close(); err != nil {
		return err
	}
	// if the checksums match, we didn't make any changes, so we can skip the
	// rename and avoid the mtime/etc update of the file.
	if hIn.Sum32() != hOut.Sum32() {
		if err := os.Rename(out.Name(), fileName); err != nil {
			_ = os.Remove(out.Name())
			return err
		}
	} else {
		// we didn't make any changes, so just remove the temp file
		if err := os.Remove(out.Name()); err != nil {
			return err
		}
	}
	return nil
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
