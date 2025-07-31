package apt

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"fastcat.org/go/gdev/sys"
)

type AptSourceInstaller struct {
	SourceName string
	Source     *AptSource
	SigningKey []byte
}

// Returns true if anything changed, false if the source was already installed.
func (i *AptSourceInstaller) Install822(ctx context.Context) (bool, error) {
	if err := i.validate(); err != nil {
		return false, err
	}
	var content bytes.Buffer
	if err := FormatDeb822(i.Source.ToDeb822(), &content); err != nil {
		return false, fmt.Errorf("failed to format deb822 for %q: %w", i.SourceName, err)
	}
	srcName := filepath.Join("/etc/apt/sources.list.d", i.SourceName+".sources")
	return i.install(ctx, srcName, &content)
}

func (i *AptSourceInstaller) install(
	ctx context.Context,
	filename string,
	content *bytes.Buffer,
) (bool, error) {
	// check if the file already has the same content
	// TODO: make this a semantic comparison
	existing, err := os.ReadFile(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			return false, fmt.Errorf("failed to read existing source file %q: %w", filename, err)
		}
	} else if bytes.Equal(existing, content.Bytes()) {
		return false, nil
	}

	if err := sys.WriteFileAsRoot(ctx, filename, content, 0o644); err != nil {
		return true, fmt.Errorf("failed to write source file %q: %w", filename, err)
	}

	// if we have a key, write it out too
	if len(i.Source.SignedBy) > 0 {
		if err := sys.WriteFileAsRoot(ctx, i.Source.SignedBy, bytes.NewReader(i.SigningKey), 0o644); err != nil {
			return true, fmt.Errorf("failed to write signing key %q: %w", i.Source.SignedBy, err)
		}
	}

	return true, nil
}

func (i *AptSourceInstaller) InstallList(ctx context.Context) (bool, error) {
	if err := i.validate(); err != nil {
		return false, err
	}
	content := bytes.NewBuffer(i.Source.ToList())
	srcName := filepath.Join("/etc/apt/sources.list.d", i.SourceName+".list")
	return i.install(ctx, srcName, content)
}

func (i *AptSourceInstaller) validate() error {
	if i.SourceName == "" {
		return fmt.Errorf("no source name provided")
	} else if i.Source == nil {
		return fmt.Errorf("no source provided for %q", i.SourceName)
	} else if err := i.Source.validate(); err != nil {
		return fmt.Errorf("source %q is invalid: %w", i.SourceName, err)
	} else if i.Source.SignedBy != "" && len(i.SigningKey) == 0 {
		return fmt.Errorf("source %q has signed-by set but no signing key provided", i.SourceName)
	}
	// TODO: check if key file extension matches key bytes content

	return nil
}
