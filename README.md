# go-build-template

Opinionated template for Golang projects.

* Based on https://github.com/thockin/go-build-template
* Requires `docker` and `docker-compose` to use.
* Uses `docker-compose` rather than raw docker to allow adding services during
  development and having compose link them together.
* Uses `dep` for dependency manageement
* Includes `cobra` for cli parsing

## Usage

To use, copy these files and make the following changes:

In the Makefile:

* change BIN to your binary name
* change PKG to the Go import path of the repo
* change REGISTRY to the Docker registry id you wish to push to.

In Dockerfile.in:

* change MAINTAINER if required
* maybe change or remove USER if required

Finally:

* Rename `cmd/myapp` to `cmd/$BIN`

## Building

Run `make` or `make build` to build our binary compiled for `linux/amd64`
with the current directory volume mounted into place. This will store
incremental state for the fastest possible build. To build for `arm` or
`arm64` you can use: `make build ARCH=arm` or `make build ARCH=arm64`. To
build all architectures you can run `make all-build`.

Run `make container` to package the binary inside a container. It will
calculate the image tag based on the current VERSION (calculated from git tag
or commit - see `make version` to view the current version). To build
containers for the other supported architectures you can run
`make container ARCH=arm` or `make container ARCH=arm64`. To make all
containers run `make all-container`.

Run `make push` to push the container image to `REGISTRY`, and similarly you
can run `make push ARCH=arm` or `make push ARCH=arm64` to push different
architecture containers. To push all containers run `make all-push`.

Run `make clean` to clean up.
