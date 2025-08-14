package apt

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"fastcat.org/go/gdev/sys"
)

type SourceInstaller struct {
	SourceName string
	Source     *Source
	SigningKey []byte
	Deb822     bool
}

func (i *SourceInstaller) AsDeb822() *SourceInstaller {
	i.Deb822 = true
	return i
}

func (i *SourceInstaller) AsList() *SourceInstaller {
	i.Deb822 = false
	return i
}

// Returns true if anything changed, false if the source was already installed.
func (i *SourceInstaller) Install(ctx context.Context) (bool, error) {
	filename, content, err := i.prepare()
	if err != nil {
		return false, err
	}
	return i.install(ctx, filename, content, false)
}

// Returns true if anything would be changed, false if the source was already installed.
func (i *SourceInstaller) Sim(ctx context.Context) (bool, error) {
	filename, content, err := i.prepare()
	if err != nil {
		return false, err
	}
	return i.install(ctx, filename, content, true)
}

func (i *SourceInstaller) prepare() (filename string, content *bytes.Buffer, err error) {
	if err := i.validate(); err != nil {
		return "", nil, err
	}
	var ext string
	if i.Deb822 {
		content = &bytes.Buffer{}
		if err := FormatDeb822Stanza(i.Source.ToDeb822(), deb822SourcesFirstKeys, content); err != nil {
			return "", nil, fmt.Errorf("failed to format deb822 for %q: %w", i.SourceName, err)
		}
		ext = ".sources"
	} else {
		content = bytes.NewBuffer(i.Source.ToList())
		ext = ".list"
	}
	srcName := filepath.Join("/etc/apt/sources.list.d", i.SourceName+ext)
	return srcName, content, nil
}

func (i *SourceInstaller) install(
	ctx context.Context,
	filename string,
	content *bytes.Buffer,
	sim bool,
) (bool, error) {
	// check if the file already has the same content
	listEq, keyEq := false, false
	if existing, err := os.ReadFile(filename); err != nil {
		if !os.IsNotExist(err) {
			return false, fmt.Errorf("failed to read existing source file %q: %w", filename, err)
		}
	} else if bytes.Equal(existing, content.Bytes()) {
		listEq = true
	} else if e822, err := ParseDeb822Stanza(bytes.NewReader(existing)); err == nil {
		if existingSrc, err := FromDeb822(e822); err == nil && i.Source.Equal(existingSrc) {
			// semantically equal, don't bother rewriting the file
			listEq = true
		}
	}
	if len(i.Source.SignedBy) > 0 {
		if existing, err := os.ReadFile(i.Source.SignedBy); err != nil {
			if !os.IsNotExist(err) {
				return false, fmt.Errorf("failed to read existing signing key %q: %w", i.Source.SignedBy, err)
			}
		} else if bytes.Equal(existing, i.SigningKey) {
			keyEq = true
		}
	} else {
		keyEq = true
	}

	if listEq && keyEq {
		fmt.Printf("APT source %s already installed\n", i.SourceName)
		return false, nil
	}

	if sim {
		if listEq {
			fmt.Printf("Would not write apt source file %s, already up to date\n", filename)
		} else {
			fmt.Printf("Would write apt source file %s\n", filename)
		}
		if keyEq {
			fmt.Printf("Would not write signing key %s, already up to date\n", i.Source.SignedBy)
		} else {
			fmt.Printf("Would write signing key %s\n", i.Source.SignedBy)
		}
		return true, nil
	}

	if !listEq {
		fmt.Printf("Writing apt source file %s\n", filename)
		if err := sys.WriteFileAsRoot(ctx, filename, content, 0o644); err != nil {
			return true, fmt.Errorf("failed to write source file %q: %w", filename, err)
		}
	}
	if !keyEq && len(i.Source.SignedBy) > 0 {
		fmt.Printf("Writing signing key %s\n", i.Source.SignedBy)
		if err := sys.WriteFileAsRoot(ctx, i.Source.SignedBy, bytes.NewReader(i.SigningKey), 0o644); err != nil {
			return true, fmt.Errorf("failed to write signing key %q: %w", i.Source.SignedBy, err)
		}
	}

	return true, nil
}

func (i *SourceInstaller) validate() error {
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
