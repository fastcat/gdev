package stack

import (
	"fmt"
	"sync/atomic"

	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/service"
)

var (
	servicesLocked           atomic.Bool
	allServices              = map[string]service.Service{}
	infraOrder, serviceOrder []string
)

func checkCanAddServices() {
	if servicesLocked.Load() {
		panic(fmt.Errorf("cannot add services after they have been locked"))
	}
}

func lockServices() {
	internal.CheckLockedDown()
	servicesLocked.Store(true)
}

func AddService(svc service.Service) {
	checkCanAddServices()
	name := svc.Name()
	if _, ok := allServices[name]; ok {
		panic(fmt.Errorf("already registered service %s", name))
	}
	allServices[name] = svc
	serviceOrder = append(serviceOrder, name)
}

func AddInfrastructure(svc service.Service) {
	checkCanAddServices()
	name := svc.Name()
	if _, ok := allServices[name]; ok {
		panic(fmt.Errorf("already registered service %s", name))
	}
	allServices[name] = svc
	infraOrder = append(infraOrder, name)
}

func AllInfrastructure() []service.Service {
	lockServices()
	ret := make([]service.Service, 0, len(infraOrder))
	for _, n := range infraOrder {
		ret = append(ret, allServices[n])
	}
	return ret
}

func AllServices() []service.Service {
	lockServices()
	ret := make([]service.Service, 0, len(serviceOrder))
	for _, n := range serviceOrder {
		ret = append(ret, allServices[n])
	}
	return ret
}

// ServiceByName returns the service with the given name, if it exists.
//
// If the service does not exist, it returns nil.
func ServiceByName(name string) service.Service {
	lockServices()
	return allServices[name]
}
