#!/bin/sh

set -e

cd "$(dirname "$0")/.."

git pull --ff-only
TASK="update and deployment" scripts/deploy.sh "$@"
