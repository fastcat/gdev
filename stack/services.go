package stack

import (
	"fmt"
	"maps"
	"slices"

	"fastcat.org/go/gdev/service"
)

var allServices = map[string]service.Service{}

func AddService(svc service.Service) {
	name := svc.Name()
	if _, ok := allServices[name]; ok {
		panic(fmt.Errorf("already registered service %s", name))
	}
	allServices[name] = svc
}

func AllServices() []service.Service {
	return slices.Collect(maps.Values(allServices))
}
