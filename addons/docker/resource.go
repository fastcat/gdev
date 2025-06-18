package docker

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/resource"
)

type containerResource struct {
	Name  string
	Image string
	Ports []string
	Env   map[string]string
}

// Container creates a new container resource with the specified name and
// options.
//
// The name will be prefixed with the instance name, e.g. if this gdev build
// calls itself xdev, then the name will be prefixed with `xdev-`.
//
// This is a convenience tool for extremely simple use cases, and should not be
// used in more complex scenarios.
func Container(
	name, image string,
	ports []string,
	env map[string]string,
) *containerResource {
	return &containerResource{
		Name:  name,
		Image: image,
		Ports: slices.Clone(ports),
		Env:   maps.Clone(env),
	}
}

// ContainerImages implements resource.ContainerResource.
func (c *containerResource) ContainerImages(context.Context) ([]string, error) {
	return []string{c.Image}, nil
}

// ID implements resource.ContainerResource.
func (c *containerResource) ID() string {
	return "docker/container/" + c.Name
}

// Ready implements resource.ContainerResource.
func (c *containerResource) Ready(context.Context) (bool, error) {
	// TODO
	return true, nil
}

// Start implements resource.ContainerResource.
func (c *containerResource) Start(ctx context.Context) error {
	cli := resource.ContextValue[client.APIClient](ctx)
	if cli == nil {
		return fmt.Errorf("docker client not found in context")
	}
	cc := container.Config{
		Image: c.Image,
	}
	if len(c.Env) > 0 {
		envs := make([]string, 0, len(c.Env))
		for k, v := range c.Env {
			envs = append(envs, fmt.Sprintf("%s=%s", k, v))
		}
		cc.Env = envs
	}
	hc := container.HostConfig{}
	if len(c.Ports) > 0 {
		exposed, bindings, err := nat.ParsePortSpecs(c.Ports)
		if err != nil {
			return fmt.Errorf("failed to parse port specs %v: %w", c.Ports, err)
		}
		cc.ExposedPorts = exposed
		hc.PortBindings = bindings
	}
	cr, err := cli.ContainerCreate(
		ctx,
		&cc,
		&hc,
		&network.NetworkingConfig{},
		nil, // platform
		c.realName(),
	)
	if err != nil {
		return fmt.Errorf("failed to create container %s: %w", c.Name, err)
	}
	err = cli.ContainerStart(ctx, cr.ID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container %s(%s): %w", c.Name, c.ID(), err)
	}
	return nil
}

// Stop implements resource.ContainerResource.
func (c *containerResource) Stop(ctx context.Context) error {
	cli := resource.ContextValue[client.APIClient](ctx)
	if cli == nil {
		return fmt.Errorf("docker client not found in context")
	}
	err := cli.ContainerRemove(
		ctx,
		c.realName(), // param is named id but accepts name too
		container.RemoveOptions{
			Force:         true,
			RemoveVolumes: true,
		},
	)
	if err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to remove container %s(%s): %w", c.Name, c.ID(), err)
	}
	return nil
}

func (c *containerResource) realName() string {
	return instance.AppName() + "-" + c.Name
}
