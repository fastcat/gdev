package shx

import (
	"bytes"
	"errors"
	"io"
	"os"
)

type outCapture struct {
	// writer, if set, is an alternate destination for output. if set, buffer will
	// not be used.
	writer io.WriteCloser
	// buffer will be used to store output if writer is not set, up to a size
	// limit. if buffer gets too big, capture will be shifted to tmpFile.
	buffer *bytes.Buffer
	// tmpFile, if set, is a temporary file that will be used to store output. it
	// is only used if writer is not set and buffer got too large.
	tmpFile *os.File
}

func (c *outCapture) init() {
	if c.writer != nil || c.tmpFile != nil || c.buffer != nil {
		return
	}
	// make sure there's a non-nil buffer to write to / read EOF from
	c.buffer = &bytes.Buffer{}
}

func (c *outCapture) Close() error {
	if c == nil {
		return nil
	}
	var errs []error
	if c.writer != nil {
		errs = append(errs, c.writer.Close())
		c.writer = nil
	}
	c.buffer = nil
	if c.tmpFile != nil {
		errs = append(errs, c.tmpFile.Close())
		c.tmpFile = nil
	}
	return errors.Join(errs...)
}

const outCapMaxBuffer = 1024 * 1024 // 1MB

func (c *outCapture) switchToTmp() error {
	var err error
	c.tmpFile, err = os.CreateTemp("", "shx-out-capture-")
	if err != nil {
		return err
	}
	// unlink the file so it's inaccessible, and we don't have to worry about it
	// later
	if err := os.Remove(c.tmpFile.Name()); err != nil {
		if err2 := c.tmpFile.Close(); err2 != nil {
			err = errors.Join(err, err2)
		}
		c.tmpFile = nil
		return err
	}
	if _, err := c.tmpFile.Write(c.buffer.Bytes()); err != nil {
		if err2 := c.tmpFile.Close(); err2 != nil {
			err = errors.Join(err, err2)
		}
		c.tmpFile = nil
		return err
	}
	c.buffer = nil
	return nil
}

func (c *outCapture) Write(p []byte) (n int, err error) {
	if c.writer != nil {
		return c.writer.Write(p)
	}
	if c.buffer != nil && c.buffer.Len()+len(p) > outCapMaxBuffer {
		if err := c.switchToTmp(); err != nil {
			return 0, err
		}
	}
	if c.tmpFile != nil {
		return c.tmpFile.Write(p)
	}
	if c.buffer == nil {
		c.buffer = &bytes.Buffer{}
	}
	return c.buffer.Write(p)
}

func (c *outCapture) doneWriting() error {
	if c == nil {
		return nil
	}
	if c.writer != nil {
		if err := c.writer.Close(); err != nil {
			return err
		}
		c.writer = nil
	}
	if c.tmpFile != nil {
		// seek to the start so we can read it back out
		if _, err := c.tmpFile.Seek(0, io.SeekStart); err != nil {
			return err
		}
	}
	return nil
}

// reader returns a reader over the captured output. If the output was written
// to a custom writer, this returns nil.
//
// doneWriting must be called before this can be used.
//
// Calling with a nil receiver returns nil.
func (c *outCapture) reader() io.Reader {
	if c == nil {
		return nil
	}
	if c.writer != nil {
		return nil
	}
	if c.tmpFile != nil {
		// need to have called doneWriting before we can read from tmpFile
		return c.tmpFile
	}
	if c.buffer != nil {
		return c.buffer
	}
	return nil // no output captured
}
