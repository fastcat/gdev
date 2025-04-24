package client

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

type HTTPError struct {
	Resp *http.Response
	Body []byte
	Err  error
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("%v: http response status %d", e.Err, e.Resp.StatusCode)
}

func heFromResp(
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
