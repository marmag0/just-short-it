# Log helpers, flag handling, and .env lookup shared by the scripts. Source it from the repository root.

errors=0
hints=0
findings=

step() { printf '[*] %s\n' "$1"; }
ok() { printf '[+] %s\n' "$1"; }
err() {
	errors=$((errors + 1))
	finding "[!] $1"
}
info() {
	hints=$((hints + 1))
	finding "[i] $1"
}

# finding prints a line and keeps it for the summary.
finding() {
	printf '%s\n' "$1"
	findings=$(printf '%s\n    %s' "$findings" "$1")
}

# summary prints the colored result of task $1, lists the findings, and fails when there were errors.
summary() {
	if [ "$errors" -gt 0 ]; then
		printf '\033[31m[!] %s error(s), %s recommendation(s):%s\033[0m\n' "$errors" "$hints" "$findings"
		return 1
	elif [ "$hints" -gt 0 ]; then
		printf '\033[33m[i] no errors, %s recommendation(s):%s\033[0m\n' "$hints" "$findings"
	else
		printf '\033[32m[+] %s successful\033[0m\n' "$1"
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
