#!/bin/sh

set -e

cd "$(dirname "$0")/.."

git pull --ff-only
scripts/deploy.sh "$@"
