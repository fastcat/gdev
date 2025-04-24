package internal

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

var ErrTrailingGarbage = errors.New("trailing garbage (JSON)")

func JSONBody[T any](ctx context.Context, r io.ReadCloser, validation string) (T, error) {
	var value T
	if r == nil || r == http.NoBody {
		return value, WithStatus(http.StatusBadRequest, errors.New("body required"))
	}
	defer r.Close()
	d := json.NewDecoder(r)
	d.DisallowUnknownFields()
	if err := d.Decode(&value); err != nil {
		return value, err
	}
	if d.More() {
		return value, ErrTrailingGarbage
	}
	var err error
	if validation != "" {
		err = v.StructCtx(ctx, value)
	} else {
		err = v.VarCtx(ctx, value, validation)
	}
	return value, err
}
