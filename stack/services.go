package stack

import (
	"fmt"

	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/service"
)

var (
	allServices              = map[string]service.Service{}
	infraOrder, serviceOrder []string
)

func AddService(svc service.Service) {
	internal.CheckCanCustomize()
	name := svc.Name()
	if _, ok := allServices[name]; ok {
		panic(fmt.Errorf("already registered service %s", name))
	}
	allServices[name] = svc
	serviceOrder = append(serviceOrder, name)
}

func AddInfrastructure(svc service.Service) {
	internal.CheckCanCustomize()
	name := svc.Name()
	if _, ok := allServices[name]; ok {
		panic(fmt.Errorf("already registered service %s", name))
	}
	allServices[name] = svc
	infraOrder = append(infraOrder, name)
}

func AllInfrastructure() []service.Service {
	internal.CheckLockedDown()
	ret := make([]service.Service, 0, len(infraOrder))
	for _, n := range infraOrder {
		ret = append(ret, allServices[n])
	}
	return ret
}

func AllServices() []service.Service {
	internal.CheckLockedDown()
	ret := make([]service.Service, 0, len(serviceOrder))
	for _, n := range serviceOrder {
		ret = append(ret, allServices[n])
	}
	return ret
}
