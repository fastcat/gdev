package apt

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"iter"
)

func AscToGPG(armored io.Reader, binary io.Writer) error {
	r := seqReader(unwrapArmor(armored))
	defer r.Close() //nolint:errcheck
	br := base64.NewDecoder(base64.RawStdEncoding, r)
	_, err := io.Copy(binary, br)
	return err
}

var (
	fiveDashes   = []byte("-----")
	fiveDashesLF = []byte("-----\n")
	colonSpace   = []byte(": ")
)

// see https://www.rfc-editor.org/rfc/rfc4880
//
// this is not a full implementation, just good enough to extract the base64
func unwrapArmor(armored io.Reader) iter.Seq2[[]byte, error] {
	br := bufio.NewReader(armored)
	sawHeader, sawBlank, sawFooter := false, false, false
	return func(yield func([]byte, error) bool) {
		for {
			line, err := br.ReadBytes('\n')
			emit := false
			if !sawHeader {
				if len(line) > 2*len(fiveDashes) &&
					bytes.HasPrefix(line, fiveDashes) &&
					bytes.HasSuffix(line, fiveDashesLF) {
					sawHeader = true
				} else if err == nil {
					err = fmt.Errorf("unexpected line instead of GPG header")
				}
			} else if !sawBlank {
				if len(bytes.TrimSpace(line)) == 0 {
					sawBlank = true
				} else if len(line) >= 4 && bytes.Contains(line, colonSpace) {
					// probably a header, don't need to check it closely
				} else if err == nil {
					err = fmt.Errorf("unexpected line looking for GPG headers/blank")
				}
			} else if !sawFooter {
				if len(line) > 2*len(fiveDashes) &&
					bytes.HasPrefix(line, fiveDashes) &&
					bytes.HasSuffix(line, fiveDashesLF) {
					sawFooter = true
				} else {
					// it's a base64 line
					emit = true
				}
			} else if len(line) == 0 && errors.Is(err, io.EOF) {
				yield(nil, io.EOF)
				return
			} else {
				err = fmt.Errorf("trailing garbage after GPG footer")
			}
			if emit {
				fmt.Printf("unwrapped: %q\n", line)
				if !yield(line, err) {
					return
				}
			} else if err != nil {
				yield(nil, err)
				return
			}
		}
	}
}

func seqReader(src iter.Seq2[[]byte, error]) io.ReadCloser {
	next, stop := iter.Pull2(src)
	return &pullReader{
		next: next,
		stop: stop,
	}
}

type pullReader struct {
	next func() ([]byte, error, bool)
	stop func()
	// buf is any extra data from a prior call to next
	buf []byte
	// err is any error from a prior call to next when the caller didn't read to
	// the point of the error
	err error
}

// Read implements io.Reader.
func (r *pullReader) Read(p []byte) (n int, err error) {
	if r.next == nil {
		return 0, io.ErrClosedPipe
	}
	rem := len(p)
	for rem > 0 {
		if len(r.buf) == 0 {
			// once we consume the buffer, if there's an error, return it
			if r.err != nil {
				return n, r.err
			}
			var ok bool
			r.buf, r.err, ok = r.next()
			if !ok {
				r.buf, r.err = nil, io.EOF
				return n, io.EOF
			}
		}
		if bl := len(r.buf); bl > 0 {
			if bl >= rem {
				copy(p[n:], r.buf[:rem])
				r.buf = r.buf[rem:]
				n += rem
				// don't return error here even if bl==rem, wait for a read that we
				// can't fully satisfy to report a read failure / EOF
				return n, nil
			}
			// 0 < bl < rem
			copy(p[n:], r.buf)
			n += bl
			rem -= bl
			r.buf = nil
		}
	}
	// don't return any stored error if we read the full amount, even if we
	// emptied the buffer. save that for the next read.
	return n, nil
}

func (r *pullReader) Close() error {
	if r.stop != nil {
		r.stop()
		r.stop = nil
	}
	r.buf = nil
	r.err = nil
	r.next = nil
	return nil
}
