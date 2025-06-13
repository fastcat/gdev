package internal

func FilterSlice[E any, S ~[]E](s S, f func(E) bool) S {
	if len(s) == 0 {
		return s
	}
	var result S
	for _, e := range s {
		if f(e) {
			result = append(result, e)
		}
	}
	return result
}
