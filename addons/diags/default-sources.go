package diags

import (
	"bytes"
	"context"
	"encoding/json"

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
	contents, err := json.Marshal(appInfo)
	if err != nil {
		// should be unreachable
		return err
	}
	return coll.Collect(ctx, "app-info.json", bytes.NewReader(contents))
}
