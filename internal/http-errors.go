package internal

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type StatusCodeErr interface {
	StatusCode() int
}

func IsNotFound(err error) bool {
	var sce StatusCodeErr
	if errors.As(err, &sce) {
		if sce.StatusCode() == http.StatusNotFound {
			return true
		}
	}
	return false
}

type HTTPError struct {
	Resp *http.Response
	Body []byte
	Err  error
}

func (e *HTTPError) Error() string {
	ct := e.Resp.Header.Get("content-type")
	if len(e.Body) > 0 && (ct == "text/plain" || ct == "application/json") {
		return fmt.Sprintf("%v: http response status %d: %s",
			e.Err,
			e.Resp.StatusCode,
			strings.TrimSpace(string(e.Body)),
		)
	}
	return fmt.Sprintf("%v: http response status %d", e.Err, e.Resp.StatusCode)
}

// StatusCode implements [internal.StatusCodeErr]
func (e *HTTPError) StatusCode() int {
	return e.Resp.StatusCode
}

func HTTPResponseErr(
	r *http.Response,
	msg string,
) *HTTPError {
	err := &HTTPError{Resp: r, Err: errors.New(msg)}
	defer r.Body.Close() //nolint:errcheck
	body, bodyErr := io.ReadAll(r.Body)
	err.Body = body
	if bodyErr != nil {
		err.Err = fmt.Errorf("%s: reading body: %w", msg, bodyErr)
	}
	return err
}

func IsHTTPOk(r *http.Response) bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}
