#!/bin/sh

set -e

cd "$(dirname "$0")/.."
. scripts/common.sh

step "Containers"
for service in $(docker compose $profile config --services); do
	state=$(docker compose $profile ps --all --format '{{.State}}' "$service")
	if [ "$state" = running ]; then
		ok "$service is running"
	else
		err "$service is ${state:-not created}"
	fi
done

# A service that has just started gets a few seconds to answer.
probe="curl --silent --output /dev/null --max-time 5 --retry 10 --retry-delay 1 --retry-all-errors"

if [ "$errors" -eq 0 ]; then
	step "Endpoints"
	port=$(env_value .env SHORTENER_PORT)
	if $probe "http://127.0.0.1:$port/"; then
		ok "shortener answers on port $port"
	else
		err "shortener does not answer on port $port"
	fi
	port=$(env_value .env GRAFANA_PORT)
	if $probe --fail "http://127.0.0.1:$port/api/health"; then
		ok "Grafana is healthy on port $port"
	else
		err "Grafana is not healthy on port $port"
	fi
fi

# deploy.sh and update.sh name the task that this check concludes.
summary "${TASK:-status check}"
