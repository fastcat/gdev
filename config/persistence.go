package config

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"strings"

	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/internal"
	"github.com/goccy/go-yaml"
)

var loadedComments yaml.CommentMap

func load() error {
	internal.CheckLockedDown()
	fn := os.ExpandEnv("${HOME}/.config/" + instance.AppName() + ".yaml")
	f, err := os.Open(fn)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil // no config file, that's ok
		}
		return err
	}
	defer f.Close()
	cm := yaml.CommentMap{}
	d := yaml.NewDecoder(f, yaml.CommentToMap(cm))
	decoded := make(map[string]any, len(keys))
	if err := d.Decode(&decoded); err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	for k := range decoded {
		if _, ok := keys[k]; !ok {
			return fmt.Errorf("unknown config key %q", k)
		}
	}
	// don't change the in-memory config until we get everything OK
	newValues := make(map[string]any, len(decoded))
	for k, v := range decoded {
		vv, err := keys[k].newFrom(v)
		if err != nil {
			return fmt.Errorf("error loading config key %q: %w", k, err)
		}
		newValues[k] = vv
	}

	maps.Copy(data, newValues)

	return nil
}

func Save() error {
	internal.CheckLockedDown()
	if data == nil {
		return fmt.Errorf("config not initialized")
	}
	fn := os.ExpandEnv("${HOME}/.config/" + instance.AppName() + ".yaml.tmp")
	f, err := os.Create(fn)
	if err != nil {
		return fmt.Errorf("error creating config temp file %q: %w", fn, err)
	}
	defer f.Close()

	// don't save default values
	toSave := maps.Clone(data)
	for k, v := range toSave {
		if keys[k].isDefault(v) {
			delete(toSave, k)
		}
	}

	// try to preserve loaded comments
	e := yaml.NewEncoder(f, yaml.WithComment(loadedComments))
	if err := e.Encode(toSave); err != nil {
		return fmt.Errorf("error writing config file %q: %w", fn, err)
	} else if err := f.Sync(); err != nil {
		return fmt.Errorf("error syncing config file %q: %w", fn, err)
	} else if err := f.Close(); err != nil {
		return fmt.Errorf("error closing config file %q: %w", fn, err)
	} else if err := os.Rename(fn, strings.TrimSuffix(fn, ".tmp")); err != nil {
		return fmt.Errorf("error renaming config file %q: %w", fn, err)
	}

	return nil
}
