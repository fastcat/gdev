package main

import (
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"golang.org/x/crypto/openpgp/armor" //nolint:staticcheck // armor parsing is fine within deprecation
)

func main() {
	// TODO: error checking and cli help
	url, filename := os.Args[1], os.Args[2]
	tmpF, err := os.CreateTemp(filepath.Dir(filename), "gdev-apt-key-*")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpF.Name()) //nolint:errcheck
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		panic("failed to download key: " + resp.Status)
	}
	in, out := io.ReadCloser(resp.Body), io.WriteCloser(tmpF)
	addLF := false
	if urlExt, fileExt := path.Ext(url), filepath.Ext(filename); urlExt == ".gpg" && fileExt == ".asc" {
		// armor
		out, err = armor.Encode(out, "PGP PUBLIC KEY BLOCK", nil)
		if err != nil {
			panic(err)
		}
		// armor.Encode doesn't write a final newline
		addLF = true
	} else if urlExt == ".asc" && fileExt == ".gpg" {
		// dearmor
		block, err := armor.Decode(in)
		if err != nil {
			panic(err)
		}
		in = io.NopCloser(block.Body)
	}
	if _, err := io.Copy(out, in); err != nil {
		panic(err)
	}

	if out != tmpF {
		if err := out.Close(); err != nil {
			panic(err)
		}
	}
	if addLF {
		if _, err := tmpF.WriteString("\n"); err != nil {
			panic(err)
		}
	}
	if err := tmpF.Sync(); err != nil {
		panic(err)
	}
	if err := tmpF.Close(); err != nil {
		panic(err)
	} else if err := os.Rename(tmpF.Name(), filename); err != nil {
		panic(err)
	}
}
