# gdev Addons

These are the "built-in" addons that gdev provides for customizing your `xdev`
tool. These are not privileged however, you can always build your own using the
same APIs.

The `gdev` code is also intentionally under a relatively permissive license
(Apache-2.0), so you can also copy & customize these if you need features
specific to your environment that aren't suitable to be "upstreamed".

The _intent_ however is that these provide sufficient customization hooks so
that forking them is rarely necessary.

## Available Addons

| Name         | Description                                                          |
| ------------ | -------------------------------------------------------------------- |
| `_template`  | Not an actual addon, but a template to copy to start a new one       |
| `asdf`       | Adds `bootstrap` step to install `asdf` and its plugins              |
| `bootstrap`  | Adds `bootstrap` command(s) to initialize developer PC               |
| `build`      | Adds support & command(s) for building source repos                  |
| `containerd` | Support for `containerd` interaction and `k3s` backend configuration |
| `containers` | Not an addon, generic container helper code for other addons         |
| `diags`      | Collect diagnostics from users for remote assistance                 |
| `docker`     | Support for interfacing with docker and using as `k3s` backend       |
| `docs`       | Command for generating documentation (man pages & markdown) for CLI  |
| `gcloud`     | Bootstrap step(s) for installing & logging in with `gcloud` CLI      |
| `gcs`        | Support for running GCS emulator and connecting to it or "real" GCS  |
| `github`     | Bootstrap steps for installing `gh` CLI and logging in               |
| `gocache`    | `GOCACHEPROG` implementations                                        |
| `golang`     | Support for building Go projects from source                         |
| `k3s`        | Support for running `k3s` locally to run your stack in Kubernetes    |
| `k8s`        | Generic support for interacting with a local Kubernetes environment  |
| `mariadb`    | Support for running MariaDB (MySQL fork) in (local) Kubernetes       |
| `nodejs`     | Support for building projects with NodeJS                            |
| `pm`         | Local process manager daemon for running things outside containers   |
| `postgres`   | Support for running PostgreSQL in (local) Kubernetes                 |
| `uv`         | Bootstrap step(s) for installing `uv` Python environment manager     |
| `valkey`     | Support for running Valkey (Redis fork) in (local) Kubernetes        |
