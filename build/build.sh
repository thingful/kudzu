#!/bin/sh

set -o errexit
set -o nounset
if set -o | grep -q "pipefail"; then
  set -o pipefail
fi

export CGO_ENABLED="${CGO_ENABLED:-0}"
export GOARCH=amd64

# generate binary assets
go generate -x "${PKG}/pkg/migrations/"

go install \
    -v \
    -installsuffix "static" \
    -ldflags "-extldflags -static -X ${PKG}/pkg/version.Version=${VERSION} -X \"${PKG}/pkg/version.BuildDate=${BUILD_DATE}\" -X ${PKG}/pkg/version.BinaryName=${BINARY_NAME}" \
    ./...
