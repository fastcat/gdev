package api

import "fastcat.org/go/gdev/internal"

type StatusCodeErr = internal.StatusCodeErr

func IsNotFound(err error) bool {
	return internal.IsNotFound(err)
}
