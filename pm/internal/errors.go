package internal

type StatusCodeErr interface {
	StatusCode() int
}

type statusCodeErr struct {
	err        error
	statusCode int
}

func (err *statusCodeErr) Error() string {
	return err.err.Error()
}

func (err *statusCodeErr) Unwrap() error {
	return err.err
}

func WithStatus(statusCode int, err error) error {
	if err == nil {
		return nil
	}
	return &statusCodeErr{err, statusCode}
}
