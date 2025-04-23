package server

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"

	"fastcat.org/go/gdev/pm/api"
)

type HTTP struct {
	Server   *http.Server
	Listener net.Listener
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

	impl := &server{
		// TODO
	}

	s := &http.Server{
		Addr:    a.String(),
		Handler: NewHTTPMux(impl),
	}
	return &HTTP{
		Server:   s,
		Listener: l,
	}, nil
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
