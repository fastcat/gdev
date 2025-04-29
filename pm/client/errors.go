package client

import (
	"net/http"

	"fastcat.org/go/gdev/internal"
)

type HTTPError = internal.HTTPError

func heFromResp(
	r *http.Response,
	msg string,
) *HTTPError {
	return internal.HTTPResponseErr(r, msg)
}
