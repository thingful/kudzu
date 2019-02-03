#!/bin/sh

set -o errexit
set -o nounset
if set -o | grep -q "pipefail"; then
  set -o pipefail
fi

export CGO_ENABLED=${CGO_ENABLED:-0}

TARGETS=$(for d in "$@"; do echo ./$d/...; done)

go test -i -installsuffix "static" ${TARGETS}
go test -v -installsuffix "static" -coverprofile=.coverage/coverage.out ${TARGETS}
go tool cover -html=.coverage/coverage.out -o .coverage/coverage.html
echo

echo -n "Checking gofmt: "
ERRS=$(find "$@" -type f -name \*.go | xargs gofmt -l 2>&1 || true)
if [ -n "${ERRS}" ]; then
    echo "FAIL - the following files need to be gofmt'ed:"
    for e in ${ERRS}; do
        echo "    $e"
    done
    echo
    exit 1
fi
echo "PASS"

echo -n "Checking go vet: "
ERRS=$(go vet ${TARGETS} 2>&1 || true)
if [ -n "${ERRS}" ]; then
    echo "FAIL"
    echo "${ERRS}"
    echo
    exit 1
fi
echo "PASS"
