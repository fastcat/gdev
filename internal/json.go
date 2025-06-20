package internal

import (
	"encoding/json"
	"errors"
	"io"
	"os"
)

var ErrTrailingGarbage = errors.New("trailing garbage in JSON file")

func ReadJSONFile[T any](name string) (T, error) {
	var result T
	f, err := os.Open(name)
	if err != nil {
		return result, err
	}
	defer f.Close()
	d := json.NewDecoder(f)
	err = d.Decode(&result)
	if err != nil {
		return result, err
	}
	if _, err = d.Token(); !errors.Is(err, io.EOF) {
		return result, ErrTrailingGarbage
	}
	return result, nil
}
