#!/bin/sh
set -e

(
  cd "$(dirname "$0")"
  go build -mod=vendor -o /tmp/gosh app/*.go
)

exec /tmp/gosh "$@"
