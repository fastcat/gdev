package textedit

import "iter"

func empty() iter.Seq[string] {
	return func(func(string) bool) {}
}

func each(v ...string) iter.Seq[string] {
	return func(yield func(string) bool) {
		for _, e := range v {
			if !yield(e) {
				break
			}
		}
	}
}
