package docker

import (
	"context"
	"fmt"
	"maps"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"fastcat.org/go/gdev/addons/containers"
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/resource"
)

type containerResource struct {
	Name       string
	Image      string
	Entrypoint []string
	Cmd        []string
	Env        map[string]string
	Ports      []string
	Mounts     []mount.Mount
}

// Container creates a new container resource with the specified name and
// options.
//
// The name will be prefixed with the instance name, e.g. if this gdev build
// calls itself xdev, then the name will be prefixed with `xdev-`.
//
// This is a convenience tool for extremely simple use cases, and should not be
// used in more complex scenarios.
func Container(name, image string) *containerResource {
	return &containerResource{
		Name:  name,
		Image: image,
	}
}

// WithEntrypoint **overwrites** the entrypoint of the container.
func (c *containerResource) WithEntrypoint(entrypoint ...string) *containerResource {
	c.Entrypoint = entrypoint
	return c
}

// WithCmd **overwrites** the command (or entrypoint args) of the container.
func (c *containerResource) WithCmd(cmd ...string) *containerResource {
	c.Cmd = append(c.Cmd, cmd...)
	return c
}

// WithEnv **appends** the environment variables to the container, or overwrites
// any existing env vars of the same names.
func (c *containerResource) WithEnv(env map[string]string) *containerResource {
	if c.Env == nil {
		c.Env = make(map[string]string, len(env))
	}
	maps.Copy(c.Env, env)
	return c
}

// WithPort **appends** the specified port(s) to the container.
func (c *containerResource) WithPorts(port ...string) *containerResource {
	c.Ports = append(c.Ports, port...)
	return c
}

func (c *containerResource) WithMounts(mounts ...mount.Mount) *containerResource {
	c.Mounts = append(c.Mounts, mounts...)
	return c
}

func (c *containerResource) WithVolumeMount(name, path string) *containerResource {
	c.Mounts = append(c.Mounts, mount.Mount{
		Type:   mount.TypeVolume,
		Source: name,
		Target: path,
	})
	return c
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
		Image:  c.Image,
		Labels: containers.DefaultLabels(),
	}
	if len(c.Cmd) > 0 {
		cc.Cmd = c.Cmd
	}
	if len(c.Entrypoint) > 0 {
		cc.Entrypoint = c.Entrypoint
	}
	if len(c.Env) > 0 {
		envs := make([]string, 0, len(c.Env))
		for k, v := range c.Env {
			envs = append(envs, fmt.Sprintf("%s=%s", k, v))
		}
		cc.Env = envs
	}
	hc := container.HostConfig{}
	if len(c.Mounts) > 0 {
		hc.Mounts = append(hc.Mounts, c.Mounts...)
	}
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
