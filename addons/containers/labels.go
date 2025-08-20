package containers

import (
	"strings"
	"sync"
	"time"

	"fastcat.org/go/gdev/instance"
)

var LabelDomain = sync.OnceValue(func() string {
	return instance.AppName() + ".gdev.fastcat.org"
})

var LabelInstance = sync.OnceValue(func() string {
	return LabelDomain() + "/instance"
})

var LabelCreatedAt = sync.OnceValue(func() string {
	return LabelDomain() + "/created-at"
})

var timeFormat = strings.ReplaceAll(time.RFC3339Nano, ":", "-")

func DefaultLabels() map[string]string {
	return map[string]string{
		LabelInstance(): instance.AppName(),
		// this causes unwanted pod restarts
		// LabelCreatedAt(): time.Now().UTC().Format(timeFormat),
	}
}
