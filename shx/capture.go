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

func (c *outCapture) Close() error {
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
