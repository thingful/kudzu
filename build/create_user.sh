#!/usr/bin/env bash

set -euo pipefail

body="{\"User\":{\"Identifier\":\"$1\",\"Provider\":\"parrot\",\"AccessToken\":\"$2\"}}"

curl -X POST \
  -d "$body" \
  "http://localhost:3001/user/new"
