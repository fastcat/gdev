package gcs

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

func NewEmulatorClient(ctx context.Context) (*storage.Client, error) {
	// replicate what the client does with env STORAGE_EMULATOR_HOST, but avoiding
	// the concurrency issues with temporarily setting that env var.
	client, err := storage.NewClient(ctx,
		option.WithoutAuthentication(),
		option.WithEndpoint(fmt.Sprintf("http://localhost:%d/storage/v1/", addon.Config.ExposedPort)),
	)
	if err != nil {
		return nil, err
	}
	return client, nil
}
