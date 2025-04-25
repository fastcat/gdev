package api

import (
	"errors"
	"net/http"
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
