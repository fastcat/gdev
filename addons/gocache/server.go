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
)

type server struct {
	impl  ReadStorage
	in    *bufio.Reader
	inDec *json.Decoder
	out   *bufio.Writer
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
	}
}

func (s *server) Run(ctx context.Context) error {
	// write caps
	writeImpl, _ := s.impl.(WriteStorage)
	caps := []Cmd{CmdClose, CmdGet}
	if writeImpl != nil {
		caps = append(caps, CmdPut)
	}
	if err := s.writeResp(&Response{ID: 0, KnownCommands: caps}); err != nil {
		return err
	}
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
			err := s.impl.Close()
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "done, hits: %v\n", hits)
			if err := s.writeResp(&Response{ID: req.ID}); err != nil {
				return err
			}
			return nil // stop server / close connection
		case CmdGet:
			resp, err := s.impl.Get(ctx, req)
			if err != nil {
				return err
			}
			if err := s.writeResp(resp); err != nil {
				return err
			}
		case CmdPut:
			if writeImpl == nil {
				return errors.New("put command not supported by storage backend")
			}
			if req.BodySize > 0 {
				body, err := s.readBody()
				if err != nil {
					return err
				}
				req.Body = body
			}
			resp, err := writeImpl.Put(ctx, req)
			if err != nil {
				return err
			}
			if err := s.writeResp(resp); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown command %q", req.Command)
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

func (s *server) readBody() (io.Reader, error) {
	// TODO: stream, pool
	var body []byte
	err := s.inDec.Decode(&body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(body), nil
}
