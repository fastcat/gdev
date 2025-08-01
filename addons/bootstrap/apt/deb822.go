package apt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
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

// ParseDeb822 parses a deb822 formatted input and returns a map of keys to
// values.
//
// It does not support continuation or multi-line values or multi-stanza files
// yet, but it will generally detect them and return an error.
//
// See https://manpages.debian.org/bookworm/dpkg-dev/deb822.5.en.html
func ParseDeb822(in io.Reader) (map[string]string, error) {
	b := bufio.NewReader(in)
	content := make(map[string]string)
	for {
		line, err := b.ReadString('\n')
		if len(line) == 0 {
			if errors.Is(err, io.EOF) {
				return content, nil
			} else if err != nil {
				return content, err
			} else {
				return content, fmt.Errorf("unexpected empty line, only one stanza supported")
			}
		}
		if line[0] == '#' {
			// comment
			continue
		}
		if line[0] >= utf8.RuneSelf {
			return content, fmt.Errorf("keys must be ASCII: %q", line)
		} else if unicode.IsSpace(rune(line[0])) {
			return content, fmt.Errorf("continuation lines not supported: %q", line)
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return content, fmt.Errorf("invalid line %q, expected key: value", line)
		}
		// key must not contain whitespace
		if strings.ContainsFunc(key, unicode.IsSpace) {
			return content, fmt.Errorf("invalid key %q, must not contain whitespace", key)
		}
		// whitespace is not significant except in multiline fields, which we don't support yet
		value = strings.TrimSpace(value)
		if _, ok := content[key]; ok {
			return content, fmt.Errorf("duplicate key %q", key)
		}
		content[key] = value
	}
}
