package diags

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	appConfig "fastcat.org/go/gdev/config"
	"fastcat.org/go/gdev/instance"
)

func CollectAppInfo(ctx context.Context, coll Collector) error {
	appInfo := struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}{
		instance.AppName(),
		instance.Version(),
	}
	return CollectJSON(ctx, coll, "app-info.json", appInfo)
}

func CollectAppConfig(ctx context.Context, coll Collector) error {
	fn := appConfig.FileName()
	f, err := os.Open(fn)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return coll.AddError(ctx, filepath.Base(fn), err)
	}
	defer f.Close() //nolint:errcheck

	return coll.Collect(ctx, filepath.Base(fn), f)
}
