#!/bin/sh

set -e

cd "$(dirname "$0")/.."
. scripts/common.sh

# --volumes also removes the stored logs, traces, and Grafana data.
case ${1:-} in
'') step "Stopping the stack" ;;
--volumes) step "Stopping the stack and removing its volumes" ;;
*)
	echo "usage: $0 [--volumes]" >&2
	exit 2
	;;
esac

if docker compose --profile tunnel down "$@"; then
	ok "the stack is down"
else
	err "docker compose down failed"
fi

summary "shutdown"
