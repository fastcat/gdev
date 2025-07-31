package apt

import (
	"fmt"
	"io"
	"slices"
)

var deb822FirstKeys = []string{
	"Types",
	"URIs",
	"Suites",
	"Components",
	"Architectures",
	"Signed-By",
}

var deb822FirstKeySet = func() map[string]struct{} {
	s := make(map[string]struct{}, len(deb822FirstKeys))
	for _, key := range deb822FirstKeys {
		s[key] = struct{}{}
	}
	return s
}()

func FormatDeb822(content map[string]string, out io.Writer) error {
	// TODO: might need line wrapping

	// write things in a predictable order
	for _, key := range deb822FirstKeys {
		if value, ok := content[key]; ok && value != "" {
			if _, err := fmt.Fprintf(out, "%s: %s\n", key, value); err != nil {
				return err
			}
		}
	}
	var extraKeys []string
	for key, value := range content {
		if _, ok := deb822FirstKeySet[key]; !ok && value != "" {
			extraKeys = append(extraKeys, key)
		}
	}
	slices.Sort(extraKeys)
	for _, key := range extraKeys {
		if _, err := fmt.Fprintf(out, "%s: %s\n", key, content[key]); err != nil {
			return err
		}
	}
	return nil
}
