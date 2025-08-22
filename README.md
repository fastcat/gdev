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
- Useful helper tools to support the above

Typically you would create your own `xdev` tools that pulls in `gdev` and the
addons your stack needs. Your `main()` would configure those addons, and any
custom ones, and then call the `gdev/cmd.Main()` to run the app. Several
examples of how to build such apps are available under `examples`, including
`examples/gdev` which has (nearly) all addons enabled, but only has
infrastructure, not app, services defined.

## Infrastructure

`gdev` provides building blocks for several common pieces of infrastructure for
running your stack:

- Docker
- Containerd
- Kubernetes
- K3S as the Kubernetes environment
  - Using either containerd or Docker as the backend
- A generic process manager daemon for running & monitoring local processes
- Databases and other "infrastructure" services to support your app
- Recipes for how to compile your code before running it when running from local
  source (instead of pre-built containers)
  - And APIs for you to provide custom recipes

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
  - using `rush` (<https://rushjs.io>)
- _More to come soon_

More will be added, and like the infrastructure services, these capabilities are
implemented using only exported APIs, so you can also add your own custom build
patterns.

## Helper Tools

### `GOBUILDCACHE`

`gdev` contains a set of addons to provide a `GOBUILDCACHE` implementation with
pluggable remote backends. Included are backends for HTTP, GCS, S3, and SFTP.
You can also provide your own backends by registering implementations of some
interfaces.

### Diags

A "diags" addon is available to help collect information from your users on
demand to diagnose problems with their development environment. A baseline set
of collectors is provided, along with an API for you to provide additional
collectors.

The result is a `.tgz` of the various files & collected data. Where this goes is
up to you. The addon comes with code to simply store it locally in a temp file
that the user can send manually, but an API is provided so you could, for
example, upload it to a storage bucket or other cloud destination to simplify
the workflow.

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
stack? No worries, your binary won't be bloated by its large client SDK. Addons
with large dependencies are broken out into separate modules, so your build also
won't have to download those unused modules.

## Addons

The full list of addons you can enable when building your `xdev` tool is in the
[addons README](https://github.com/fastcat/gdev/tree/main/addons)

## Usage

The general idea of how you use an `xdev` tool built with `gdev` is:

1. You download & install an initial version of `xdev`
2. You run `xdev bootstrap` to configure your machine (once)

Then each development cycle:

1. You use `xdev _FIXME_` to choose which services you are runing/debugging
2. You run `xdev start` to start the stack
3. You do your work
4. You run `xdev stop` to stop the stack

_This will be fleshed out with more capabilities soon._
