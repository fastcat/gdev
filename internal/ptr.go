package internal

func Ptr[T any](t T) *T { return &t }

func ValueOrZero[T any](p *T) T {
	if p != nil {
		return *p
	}
	var v T
	return v
}

func ValueOrDefault[T any](p *T, d T) T {
	if p != nil {
		return *p
	}
	return d
}
