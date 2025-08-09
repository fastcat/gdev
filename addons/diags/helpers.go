package diags

import (
	"bytes"
	"context"
	"encoding/json"
)

func CollectJSON(
	ctx context.Context,
	coll Collector,
	name string,
	v any,
) error {
	contents, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		// should be unreachable
		return coll.AddError(ctx, name, err)
	}
	contents = append(contents, '\n')
	return coll.Collect(ctx, name, bytes.NewReader(contents))
}
