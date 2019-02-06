#!/usr/bin/env bash

set -euo pipefail

curl -X POST \
  -d '{"User":{"Identifier":"f2048b96-f349-45a6-99be-ff623557ef31","Provider":"parrot","AccessToken":"y2WqFVltC07ljuNivLOJgXOCmcIEeU7fj8G7ttEUdVNQBt02","RefreshToken":"uIyoYwvI9cODJTg3wyuk6ggjS9YkqdoykjvMMbYnSFNHpIWz"}}' \
  "http://localhost:3001/user/new"
