package apt

import (
	"io"

	"golang.org/x/crypto/openpgp/armor" //nolint:staticcheck // armor parsing is fine within deprecation
)

func AscToGPG(armored io.Reader, binary io.Writer) error {
	block, err := armor.Decode(armored)
	if err != nil {
		return err
	}
	_, err = io.Copy(binary, block.Body)
	return err
}
