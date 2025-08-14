package apt

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

var deb822SourcesFirstKeys = []string{
	"Types",
	"URIs",
	"Suites",
	"Components",
	"Architectures",
	"Signed-By",
}

func FormatDeb822Stanza(
	content map[string]string,
	firstKeys []string,
	out io.Writer,
) error {
	// TODO: might need line wrapping

	// write things in a predictable order
	for _, key := range firstKeys {
		if value, ok := content[key]; ok && value != "" {
			if _, err := fmt.Fprintf(out, "%s: %s\n", key, value); err != nil {
				return err
			}
		}
	}
	firstKeySet := make(map[string]struct{}, len(firstKeys))
	for _, key := range firstKeys {
		firstKeySet[key] = struct{}{}
	}
	var extraKeys []string
	for key, value := range content {
		if _, ok := firstKeySet[key]; !ok && value != "" {
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

// ParseDeb822Stanza parses a deb822 formatted stanza and returns a map of keys
// to values.
//
// It does not support continuation or multi-line values or multi-stanza files
// yet, but it will generally detect them and return an error.
//
// See https://manpages.debian.org/bookworm/dpkg-dev/deb822.5.en.html
func ParseDeb822Stanza(in io.Reader) (map[string]string, error) {
	b := bufio.NewReader(in)
	content := make(map[string]string)
	var lastKey string
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
			if lastKey == "" {
				return content, fmt.Errorf("continuation line without predecessor: %q", line)
			}
			// replace whatever leading whitespace started the continuation line with
			// a normal space
			content[lastKey] += " " + strings.TrimSpace(line)
			continue
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
		lastKey = key
	}
}

var doubleNewline = []byte{'\n', '\n'}

// Deb822SplitStanza is a bufio.SplitFunc that splits on double newlines,
// suitable for use with bufio.Scanner to split deb822 files into stanzas.
func Deb822SplitStanza(data []byte, atEOF bool) (advance int, token []byte, err error) {
	i := bytes.Index(data, doubleNewline)
	if i < 0 {
		if atEOF && len(data) > 0 {
			// last stanza
			return len(data), data, nil
		}
		// need more data
		return 0, nil, nil
	}
	// return stanza with just the first newline, skip the second
	return i + 2, data[:i+1], nil
}
