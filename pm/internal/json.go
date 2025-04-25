package internal

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

var ErrTrailingGarbage = errors.New("trailing garbage (JSON)")

type BadResponseError struct {
	err error
}

func (err *BadResponseError) Error() string {
	return "bad response: " + err.err.Error()
}

func (err *BadResponseError) Unwrap() error { return err.err }

func badReqOrResp(err error, response bool) error {
	if err == nil {
		return nil
	} else if response {
		return &BadResponseError{err}
	} else {
		return WithStatus(http.StatusBadRequest, err)
	}
}

func JSONBody[T any](
	ctx context.Context,
	r io.ReadCloser,
	validation string,
	response bool,
) (T, error) {
	var value T
	if r == nil || r == http.NoBody {
		return value, badReqOrResp(errors.New("body required"), response)
	}
	defer r.Close() // nolint:errcheck
	d := json.NewDecoder(r)
	d.DisallowUnknownFields()
	if err := d.Decode(&value); err != nil {
		return value, err
	}
	if d.More() {
		return value, ErrTrailingGarbage
	}
	var err error
	if validation == "" {
		err = v.StructCtx(ctx, value)
	} else {
		err = v.VarCtx(ctx, value, validation)
	}
	if err != nil {
		err = badReqOrResp(err, response)
	}
	return value, err
}
