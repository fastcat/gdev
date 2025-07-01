package containers

import (
	"time"

	"fastcat.org/go/gdev/instance"
)

const (
	LabelInstance  = "fastcat.org/go/gdev/instance"
	LabelCreatedAt = "fastcat.org/go/gdev/created-at"
)

func DefaultLabels() map[string]string {
	return map[string]string{
		LabelInstance:  instance.AppName(),
		LabelCreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
}
