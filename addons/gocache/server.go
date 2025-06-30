package gocache

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/sync/errgroup"
)

type server struct {
	impl     ReadStorage
	in       *bufio.Reader
	inDec    *json.Decoder
	out      *bufio.Writer
	bodyPool sync.Pool
}

func NewServer(
	impl ReadStorage,
	in io.Reader,
	out io.Writer,
) *server {
	if impl == nil {
		panic("impl must not be nil")
	}
	bin := bufio.NewReader(in)
	d := json.NewDecoder(bin)
	return &server{
		impl:  impl,
		in:    bin,
		inDec: d,
		out:   bufio.NewWriter(out),
		bodyPool: sync.Pool{
			New: func() any {
				b := make([]byte, 0, 64*1024)
				return &b
			},
		},
	}
}

func (s *server) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// write caps
	writeImpl, _ := s.impl.(WriteStorage)
	caps := []Cmd{CmdClose, CmdGet}
	if writeImpl != nil {
		caps = append(caps, CmdPut)
	}
	if err := s.writeResp(&Response{ID: 0, KnownCommands: caps}); err != nil {
		return err
	}
	respCh := make(chan *Response, 1)
	eg1, eg1Ctx := errgroup.WithContext(ctx)
	eg1.Go(func() error {
		return s.respWriterLoop(eg1Ctx, respCh)
	})
	eg2, eg2Ctx := errgroup.WithContext(ctx)
	hits := map[Cmd]int{}

	for {
		req, err := s.readReq()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil // client closed connection, should have sent CmdClose first
			}
			return err
		}
		hits[req.Command]++
		switch req.Command {
		case CmdClose:
			fmt.Fprintf(os.Stderr, "done, hits: %v\n", hits)
			// wait for outstanding requests to complete
			var errs []error
			errs = append(errs, eg2.Wait())
			errs = append(errs, s.impl.Close())
			select {
			case <-ctx.Done():
				return fmt.Errorf("close canceled: %w", ctx.Err())
			case respCh <- &Response{ID: req.ID}:
			}
			close(respCh)
			errs = append(errs, eg1.Wait())
			return errors.Join(errs...)
		case CmdGet:
			eg2.Go(func() error {
				resp, err := s.impl.Get(eg2Ctx, req)
				if err != nil {
					return err
				}
				select {
				case <-eg2Ctx.Done():
					return fmt.Errorf("get canceled: %w", eg2Ctx.Err())
				case respCh <- resp:
					return nil
				}
			})
		case CmdPut:
			if writeImpl == nil {
				eg2.Go(func() error {
					return s.sendErrResp(eg2Ctx, respCh, req.ID, "put not supported by this server")
				})
				continue
			}
			doneBody := noop
			if req.BodySize > 0 {
				req.Body, doneBody, err = s.readBody()
				if err != nil {
					return err
				}
			}
			eg2.Go(func() error {
				defer doneBody()
				resp, err := writeImpl.Put(eg2Ctx, req)
				if err != nil {
					return err
				}
				select {
				case <-eg2Ctx.Done():
					return fmt.Errorf("put canceled: %w", eg2Ctx.Err())
				case respCh <- resp:
					return nil
				}
			})
		default:
			eg2.Go(func() error {
				return s.sendErrResp(eg2Ctx, respCh, req.ID, fmt.Sprintf("unknown command %s", req.Command))
			})
		}
	}
}

func (s *server) sendErrResp(
	ctx context.Context,
	responses chan<- *Response,
	id int64,
	msg string,
) error {
	resp := &Response{
		ID:  id,
		Err: msg,
	}
	select {
	case <-ctx.Done():
		return fmt.Errorf("err canceled: %w", ctx.Err())
	case responses <- resp:
		return nil
	}
}

func (s *server) respWriterLoop(
	ctx context.Context,
	responses <-chan *Response,
) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case resp, ok := <-responses:
			if !ok {
				return nil
			}
			if err := s.writeResp(resp); err != nil {
				return err
			}
		}
	}
}

func (s *server) writeResp(resp *Response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	if _, err := s.out.Write(data); err != nil {
		return err
	}
	if err := s.out.WriteByte('\n'); err != nil {
		return err
	}
	if err := s.out.Flush(); err != nil {
		return err
	}
	return nil
}

func (s *server) readReq() (*Request, error) {
	var req Request
	err := s.inDec.Decode(&req)
	return &req, err
}

func (s *server) readBody() (io.Reader, func(), error) {
	// TODO: stream, pool
	bodyPtr := s.bodyPool.Get().(*[]byte)
	*bodyPtr = (*bodyPtr)[:0]
	err := s.inDec.Decode(bodyPtr)
	if err != nil {
		return nil, noop, err
	}
	return bytes.NewReader(*bodyPtr),
		func() {
			*bodyPtr = (*bodyPtr)[:0]
			s.bodyPool.Put(bodyPtr)
		},
		nil
}

func noop() {}
