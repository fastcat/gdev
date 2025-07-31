package apt

import (
	"fmt"
	"io"
)

func FormatDeb822(content map[string]string, out io.Writer) error {
	// TODO: line wrapping to make things pretty?
	for key, value := range content {
		if value == "" {
			// not supported? at least not used
			continue
		}
		// TODO: there's probably some escaping rules we need to worry about in corner cases here
		if _, err := fmt.Fprintf(out, "%s: %s\n", key, value); err != nil {
			return err
		}
	}
	return nil
}
