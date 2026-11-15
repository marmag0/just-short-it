# Log helpers, flag handling, and .env lookup shared by the scripts. Source it from the repository root.

errors=0
hints=0

step() { printf '[*] %s\n' "$1"; }
ok() { printf '[+] %s\n' "$1"; }
err() {
	printf '[!] %s\n' "$1"
	errors=$((errors + 1))
}
info() {
	printf '[i] %s\n' "$1"
	hints=$((hints + 1))
}

# summary prints the colored last line and fails when there were errors.
summary() {
	if [ "$errors" -gt 0 ]; then
		printf '\033[31m[!] %s error(s), %s recommendation(s)\033[0m\n' "$errors" "$hints"
		return 1
	elif [ "$hints" -gt 0 ]; then
		printf '\033[33m[i] no errors, %s recommendation(s)\033[0m\n' "$hints"
	else
		printf '\033[32m[+] all good\033[0m\n'
	fi
}

# env_value prints the value of key $2 from the KEY=value file $1.
env_value() { sed -n "s/^$2=//p" "$1" | tail -n 1; }

# --tunnel adds cloudflared, the way in from the internet.
case ${1:-} in
'') profile= ;;
--tunnel) profile="--profile tunnel" ;;
*)
	echo "usage: $0 [--tunnel]" >&2
	exit 2
	;;
esac
