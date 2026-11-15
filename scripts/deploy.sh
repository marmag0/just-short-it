#!/bin/sh

set -e

cd "$(dirname "$0")/.."
. scripts/common.sh
tunnel_flag "$@"

scripts/prerequisites.sh "$@"
docker compose $profile up --detach --build
TASK="${TASK:-deployment}" scripts/status.sh "$@"
