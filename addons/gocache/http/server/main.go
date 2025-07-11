package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"strconv"
	"syscall"
)

func main() {
	port := 0xcace // cache without the h
	if ps := os.Getenv("PORT"); ps != "" {
		if p, err := strconv.ParseInt(ps, 0, 16); err != nil {
			panic(err)
		} else {
			port = int(p)
		}
	}
	lAddr := "localhost:" + strconv.FormatInt(int64(port), 10)
	td, err := os.MkdirTemp("", "gdev-gocache-http-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(td) //nolint:errcheck
	fmt.Println("Using temporary directory:", td)
	tdr, err := os.OpenRoot(td)
	if err != nil {
		panic(err)
	}
	defer tdr.Close() //nolint:errcheck

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	slog.SetDefault(slog.New(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}),
	))

	h := &handler{
		root: tdr,
		log:  slog.Default(),
	}

	fmt.Println("Starting HTTP server at", lAddr)
	fmt.Println("Ctrl-C to stop and cleanup")

	s := &http.Server{
		Addr:    lAddr,
		Handler: h,
	}

	go func() {
		sig := <-sigCh
		fmt.Printf("Received %v, shutting down\n", sig)
		s.Shutdown(context.TODO()) //nolint:errcheck
	}()

	if err := s.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}
}

type handler struct {
	root *os.Root
	log  *slog.Logger
}

func (h *handler) prepPath(u *url.URL) string {
	// clean the path, make it absolute, and ensure it is relative to the root
	p := path.Clean(u.Path)
	if !path.IsAbs(p) {
		// this should be impossible
		panic(fmt.Errorf("path %q is not absolute", p))
	}
	// make everything relative paths within the root, i.e. /x => ./x
	p = "." + p
	return p
}

// ServeHTTP implements http.Handler.
func (h *handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	p := h.prepPath(req.URL)
	switch req.Method {
	case http.MethodGet, http.MethodHead:
		h.getOrHead(res, req, p)
	case http.MethodPut:
		h.put(res, req, p)
	case http.MethodDelete:
		h.delete(res, req, p)
	case "MOVE":
		h.move(res, req, p)
	default:
		h.err(http.StatusMethodNotAllowed, req, fmt.Errorf(
			"unsupported method %q", req.Method,
		))
		res.WriteHeader(http.StatusMethodNotAllowed)
		res.Header().Set("Allow", "GET, HEAD, PUT, DELETE, MOVE")
	}
}

func (h *handler) errToHTTP(req *http.Request, res http.ResponseWriter, err error) {
	if errors.Is(err, os.ErrNotExist) {
		res.WriteHeader(http.StatusNotFound)
		h.err(http.StatusNotFound, req, err)
	} else if errors.Is(err, os.ErrPermission) {
		h.err(http.StatusForbidden, req, err)
		res.WriteHeader(http.StatusForbidden)
	} else if errors.Is(err, os.ErrExist) || errors.Is(err, syscall.EISDIR) {
		h.err(http.StatusConflict, req, err)
		res.WriteHeader(http.StatusConflict)
	} else {
		h.err(http.StatusInternalServerError, req, err)
		res.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *handler) getOrHead(res http.ResponseWriter, req *http.Request, p string) {
	f, err := h.root.Open(p)
	if err != nil {
		h.errToHTTP(req, res, err)
		return
	}
	defer f.Close() //nolint:errcheck
	st, err := f.Stat()
	if err != nil {
		h.err(http.StatusInternalServerError, req, err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	if st.IsDir() {
		h.err(http.StatusForbidden, req, fmt.Errorf("cannot serve directory"))
		res.WriteHeader(http.StatusForbidden)
		return
	}
	// serve everything as binary
	res.Header().Set("Content-Type", "application/octet-stream")
	// NOTE: this doesn't compute an etag, since we don't really need it
	http.ServeContent(res, req, f.Name(), st.ModTime(), f)
	// assume success?
	h.ok(http.StatusOK, req)
}

func (h *handler) put(res http.ResponseWriter, req *http.Request, p string) {
	// implicit mkdir to keep the api simple
	if err := h.root.MkdirAll(path.Dir(p), 0o700); err != nil {
		h.errToHTTP(req, res, err)
		return
	}
	f, err := h.root.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		h.errToHTTP(req, res, err)
		return
	}
	defer f.Close() //nolint:errcheck
	if _, err := io.Copy(f, req.Body); err != nil {
		h.err(http.StatusInternalServerError, req, err)
		res.WriteHeader(http.StatusInternalServerError)
	} else if err := f.Sync(); err != nil {
		h.err(http.StatusInternalServerError, req, err)
		res.WriteHeader(http.StatusInternalServerError)
	} else if err := f.Close(); err != nil {
		h.err(http.StatusInternalServerError, req, err)
		res.WriteHeader(http.StatusInternalServerError)
	} else {
		res.WriteHeader(http.StatusNoContent)
		h.ok(http.StatusNoContent, req)
	}
}

func (h *handler) delete(res http.ResponseWriter, req *http.Request, p string) {
	if err := h.root.Remove(p); err != nil {
		h.errToHTTP(req, res, err)
		return
	}
	res.WriteHeader(http.StatusNoContent)
	h.ok(http.StatusNoContent, req)
}

func (h *handler) move(res http.ResponseWriter, req *http.Request, p string) {
	dest := req.Header.Get("Destination")
	if dest == "" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	destURL, err := url.Parse(dest)
	if err != nil {
		h.err(http.StatusBadRequest, req, fmt.Errorf(
			"invalid Destination header %q: %w",
			dest, err,
		))
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	if destURL.IsAbs() {
		h.err(http.StatusBadRequest, req, fmt.Errorf(
			"invalid Destination header %q: must not be an absolute URL",
			destURL,
		))
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	dest = h.prepPath(destURL)
	if err := h.root.Rename(p, dest); err != nil {
		h.errToHTTP(req, res, err)
		return
	}
	res.WriteHeader(http.StatusNoContent)
	h.ok(http.StatusNoContent, req)
}

func (h *handler) err(status int, req *http.Request, err error) {
	h.log.Warn("Error",
		"status", status,
		"method", req.Method,
		"path", req.URL.Path,
		"err", err,
	)
}

func (h *handler) ok(status int, req *http.Request) {
	h.log.Info("OK",
		"status", status,
		"method", req.Method,
		"path", req.URL.Path,
	)
}
