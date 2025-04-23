package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"fastcat.org/go/gdev/pm/api"
)

type HTTP struct {
	Server   *http.Server
	Listener net.Listener
	daemon   *daemon
}

func NewHTTP() (*HTTP, error) {
	a := api.ListenAddr()
	if au, _ := a.(*net.UnixAddr); au != nil {
		// TODO: check if the socket is live first
		if err := os.Remove(a.String()); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	l, err := net.Listen(a.Network(), a.String())
	if err != nil {
		return nil, err
	}

	daemon := NewDaemon()

	s := &http.Server{
		Addr:    a.String(),
		Handler: NewHTTPMux(daemon),
	}
	return &HTTP{
		Server:   s,
		Listener: l,
		daemon:   daemon,
	}, nil
}

func (h *HTTP) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		log.Print("stopping pm server")
		sdCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if sdErr := h.Server.Shutdown(sdCtx); sdErr != nil {
			// force it to close harder
			_ = h.Server.Close()
		}
	}()
	err := h.Server.Serve(h.Listener)
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}
	err2 := h.daemon.terminate()
	return errors.Join(err, err2)
}

func NewHTTPMux(impl api.API) *http.ServeMux {
	m := http.NewServeMux()
	w := &httpWrapper{impl}
	m.HandleFunc("GET /{$}", w.Ping)
	return m
}

type httpWrapper struct {
	impl api.API
}

func (h *httpWrapper) Ping(w http.ResponseWriter, r *http.Request) {
	err := h.impl.Ping(r.Context())
	if err != nil {
		// TODO: status code errors
		w.Header().Set("content-type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "%v\n", err)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}
