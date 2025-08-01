package internal

import (
	"io"
	"iter"
)

// SeqReader converts an iterator to an io.ReadCloser
func SeqReader(src iter.Seq2[[]byte, error]) io.ReadCloser {
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
