# gdev

`gdev` is a toolkit for building an application for your team to help automate
and simplify your developer experience.

While `gdev` contains various runnable examples, the real use case for it is to
create your own application, built using its components, that is setup for your
team's development stack.

`gdev` understands & has components to help with:

- Setting up a new developer system ("bootstrap")
- Having a "stack" you run composed of infrastructure and application services
- Only running a subset of those services depending on what you are doing at any
  given moment
- Selecting whether to run each service from an artifact (container image,
  etc.), local source code, or in a debugger
- Building (e.g. compiling) services before starting them when running from
  local source code.

## Infrastructure

`gdev` provides building blocks for several common pieces of infrastructure:

- Docker
- Containerd
- Kubernetes
- K3S as the Kubernetes environment
  - Using either containerd or Docker as the backend
- A generic process manager daemon for running & monitoring local processes
- Postgres run under K8S
- Valkey (Redis fork) run under K8S

You can also configure your own infrastructure services. All the infrastructure
services `gdev` provides are implemented using exported APIs, access to `gdev`
internals is not required.

## Your Services

When you create your application using `gdev`, you configure the stack of your
applications you want to run. `gdev` provides helpers to make this easy,
especially for common scenarios.

As initial examples, `gdev` knows how to build some simple project types:

- Go
  - using `go build ./...`, understanding how to just build select
    sub-directories
  - using `mage` or `go tool mage`
- NodeJS
  - using `npm run build`
- _More to come soon_

More will be added, and like the infrastructure services, these capabilities are
implemented using only exported APIs, so you can also add your own custom build
patterns.

## Custom Commands

Your development environment probably benefits from some helper automation for
common tasks. `gdev` makes it easy to bundle these by allowing you to add any
custom commands you want to it. Need a DB cleanup tool to reset things between
runs? Just write some Go code for it and wire it up.

## Bloat

While `gdev` contains helper code for working with many systems, including them
all can cause major binary bloat. When building your `gdev`-based command, the
`addon` pattern means that your tool (once compiled) will only include the
dependencies needed for the components you use. Don't use Kubernetes in your
stack? No worries, your binary won't be bloated by its large client SDK.

## Usage

The general idea of how you use an "`xdev`" tool built with `gdev` is:

1. You download & install an initial version of `xdev`
2. You run `xdev bootstrap` to configure your machine (once)

Then each development cycle:

1. You use `xdev _FIXME_` to choose which services you are runing/debugging
2. You run `xdev start` to start the stack
3. You do your work
4. You run `xdev stop` to stop the stack

_This will be fleshed out with more capabilities soon._
