#!/bin/sh

set -e

cd "$(dirname "$0")/.."
. scripts/common.sh

scripts/prerequisites.sh "$@"
docker compose $profile up --detach --build
scripts/status.sh "$@"
