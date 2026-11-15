#!/bin/sh

set -e

cd "$(dirname "$0")/.."
. scripts/common.sh
tunnel_flag "$@"

step "Dependencies"
if ! command -v docker >/dev/null; then
	err "docker is not installed"
elif ! docker compose version >/dev/null 2>&1; then
	err "the docker compose plugin is not installed"
elif ! docker info >/dev/null 2>&1; then
	err "the docker daemon is not running"
else
	ok "docker with compose is running"
fi
if command -v curl >/dev/null; then
	ok "curl is installed"
else
	err "curl is not installed, status.sh needs it"
fi
for tool in git make; do
	if command -v "$tool" >/dev/null; then
		ok "$tool is installed"
	else
		info "$tool is not installed"
	fi
done

step "Configuration in .env"
if [ ! -f .env ]; then
	err ".env is missing, copy .env.example to .env and fill it in"
	summary
fi
if [ -n "$(find .env -perm -004)" ]; then
	info ".env is readable by other users, consider chmod 600 .env"
fi

# The tunnel exposes the stack, so with it an empty or publicly known secret is an error.
secret=info
[ -n "$profile" ] && secret=err

ports=
for key in $(sed -n 's/^\([A-Z_]*\)=.*/\1/p' .env.example); do
	value=$(env_value .env "$key")
	case $key in
	*_PORT)
		if ! { [ "$value" -ge 1 ] && [ "$value" -le 65535 ]; } 2>/dev/null; then
			err "$key must be a port from 1 to 65535"
			continue
		fi
		case " $ports " in
		*" $value "*) err "$key reuses port $value" ;;
		*) ok "$key=$value" ;;
		esac
		ports="$ports $value"
		;;
	*_PASSWORD | *_TOKEN)
		if [ -z "$value" ]; then
			$secret "$key is empty"
		elif [ "$value" = "$(env_value .env.example "$key")" ]; then
			$secret "$key still has the placeholder from .env.example"
		else
			ok "$key is set"
		fi
		;;
	*)
		if [ -n "$value" ]; then
			ok "$key is set"
		else
			info "$key is empty"
		fi
		;;
	esac
done

# docker compose has the final say on the variables it requires.
if [ "$errors" -eq 0 ]; then
	step "Compose configuration"
	if output=$(docker compose $profile config --quiet 2>&1); then
		ok "docker-compose.yml accepts this .env"
	else
		err "$output"
	fi
fi

summary "prerequisites check"
