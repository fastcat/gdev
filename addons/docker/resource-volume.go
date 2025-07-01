package docker

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"

	"fastcat.org/go/gdev/addons/containers"
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/resource"
)

type volumeResource struct {
	Name       string
	Driver     string
	DriverOpts map[string]string
}

// Volume creates a new volume resource with the specified name and options.
//
// The name will be prefixed with the instance name, e.g. if this gdev build
// calls itself xdev, then the name will be prefixed with `xdev-`.
//
// This is a convenience tool for extremely simple use cases, and should not be
// used in more complex scenarios.
func Volume(name string) *volumeResource {
	return &volumeResource{
		Name: name,
	}
}

func (v *volumeResource) WithDriver(driver string) *volumeResource {
	v.Driver = driver
	return v
}

func (v *volumeResource) WithDriverOpts(opts map[string]string) *volumeResource {
	if v.DriverOpts == nil {
		v.DriverOpts = make(map[string]string, len(opts))
	}
	maps.Copy(v.DriverOpts, opts)
	return v
}

// ID implements resource.Resource.
func (v *volumeResource) ID() string {
	return "docker/volume/" + v.Name
}

// Ready implements resource.Resource.
func (v *volumeResource) Ready(context.Context) (bool, error) {
	// volumes are immediately ready
	return true, nil
}

// Start implements resource.Resource.
func (v *volumeResource) Start(ctx context.Context) error {
	cli := resource.ContextValue[client.APIClient](ctx)
	if cli == nil {
		return fmt.Errorf("docker client not found in context")
	}
	_, err := cli.VolumeCreate(ctx, volume.CreateOptions{
		Name:       v.realName(),
		Labels:     containers.DefaultLabels(),
		Driver:     v.Driver,
		DriverOpts: v.DriverOpts,
	})
	if err != nil {
		if errors.Is(err, errdefs.ErrAlreadyExists) {
			// TODO: check the driver/etc match
			return nil // volume already exists, no need to create it again
		}
		return fmt.Errorf("failed to create volume %s: %w", v.Name, err)
	}
	return nil
}

// Stop implements resource.Resource.
func (v *volumeResource) Stop(context.Context) error {
	// volumes are not deleted, they represent persistent storage
	return nil
}

func (v *volumeResource) realName() string {
	return instance.AppName() + "-" + v.Name
}
