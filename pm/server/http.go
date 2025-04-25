package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"fastcat.org/go/gdev/pm/api"
	"fastcat.org/go/gdev/pm/internal"
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
	err2 := h.daemon.Terminate(ctx)
	return errors.Join(err, err2)
}

func NewHTTPMux(impl api.API) *http.ServeMux {
	m := http.NewServeMux()
	w := &httpWrapper{impl}
	reg := func(method, path string, handler http.HandlerFunc) {
		expr := method + " " + path
		if strings.HasSuffix(path, "/") {
			expr += "{$}"
		}
		m.HandleFunc(expr, handler)
	}
	reg(http.MethodGet, api.PathPing, w.Ping)
	reg(http.MethodGet, api.PathSummary, w.Summary)
	reg(http.MethodGet, api.PathOneChild, w.Child)
	reg(http.MethodPut, api.PathChild, w.PutChild)
	reg(http.MethodPost, api.PathStartChild, w.StartChild)
	reg(http.MethodPost, api.PathStopChild, w.StopChild)
	reg(http.MethodDelete, api.PathOneChild, w.DeleteChild)
	reg(http.MethodPost, api.PathTerminate, w.Terminate)
	return m
}

type httpWrapper struct {
	impl api.API
}

func (h *httpWrapper) Ping(w http.ResponseWriter, r *http.Request) {
	err := h.impl.Ping(r.Context())
	if err != nil {
		h.error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *httpWrapper) Summary(w http.ResponseWriter, r *http.Request) {
	resp, err := h.impl.Summary(r.Context())
	if err != nil {
		h.error(w, err)
		return
	}
	h.json(r, w, resp)
}

func (h *httpWrapper) Child(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue(api.PathChildParamName)
	resp, err := h.impl.Child(r.Context(), name)
	if err != nil {
		h.error(w, err)
		return
	}
	h.json(r, w, resp)
}

func (h *httpWrapper) PutChild(w http.ResponseWriter, r *http.Request) {
	body, err := internal.JSONBody[api.Child](r.Context(), r.Body, "", false)
	if err != nil {
		h.error(w, err)
		return
	}
	resp, err := h.impl.PutChild(r.Context(), body)
	if err != nil {
		h.error(w, err)
		return
	}
	h.json(r, w, resp)
}

func (h *httpWrapper) StartChild(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue(api.PathChildParamName)
	resp, err := h.impl.StartChild(r.Context(), name)
	if err != nil {
		h.error(w, err)
		return
	}
	h.json(r, w, resp)
}

func (h *httpWrapper) StopChild(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue(api.PathChildParamName)
	resp, err := h.impl.StopChild(r.Context(), name)
	if err != nil {
		h.error(w, err)
		return
	}
	h.json(r, w, resp)
}

func (h *httpWrapper) DeleteChild(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue(api.PathChildParamName)
	resp, err := h.impl.DeleteChild(r.Context(), name)
	if err != nil {
		h.error(w, err)
		return
	}
	h.json(r, w, resp)
}

func (h *httpWrapper) Terminate(w http.ResponseWriter, r *http.Request) {
	err := h.impl.Terminate(r.Context())
	if err != nil {
		h.error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *httpWrapper) error(w http.ResponseWriter, err error) {
	w.Header().Set("content-type", "text/plain")
	var sce internal.StatusCodeErr
	sc := http.StatusInternalServerError
	if errors.As(err, &sce) {
		sc = sce.StatusCode()
	}
	w.WriteHeader(sc)
	_, _ = fmt.Fprintf(w, "%v\n", err)
}

func (h *httpWrapper) json(r *http.Request, w http.ResponseWriter, body any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	e := json.NewEncoder(w)
	// humans are likely to poke this with curl, add minimal prettyness
	e.SetIndent("", " ")
	if err := e.Encode(body); err != nil {
		name := r.Pattern
		if name == "" {
			name = r.URL.Path
		}
		log.Printf("failed to write %s response: %v", name, err)
	}
}
