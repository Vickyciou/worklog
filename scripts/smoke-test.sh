#!/bin/sh
# Prove a release binary works alone in a fresh, isolated environment.
# Usage: scripts/smoke-test.sh <worklog-binary> <expected-version>
set -eu

if [ $# -ne 2 ]; then
	echo "usage: scripts/smoke-test.sh <worklog-binary> <expected-version>" >&2
	exit 2
fi
binary=$(cd "$(dirname "$1")" && pwd)/$(basename "$1")
expected_version=$2

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
export HOME="$work/home"
export XDG_CONFIG_HOME="$work/config"
export XDG_STATE_HOME="$work/state"
export GIT_CONFIG_GLOBAL="$work/gitconfig"
export GIT_CONFIG_NOSYSTEM=1
export TZ=UTC LANG=en_US.UTF-8 LC_ALL=
# The binary must not need Go, so hide any toolchain from PATH.
export PATH=$(dirname "$(command -v git)"):/usr/bin:/bin
mkdir -p "$HOME"

fail() {
	echo "smoke test failed: $*" >&2
	exit 1
}

"$binary" version | grep -q "^worklog $expected_version (" || fail "version does not report $expected_version"

repository="$HOME/project"
git init -q "$repository"
cd "$repository"
git config user.email smoke@example.invalid
git config user.name Smoke

# Profile, repository, language, journal folder, daily summary, confirm.
printf '1\n\n\n\n1\ny\n' | "$binary" setup >/dev/null || fail "setup"

git commit -q --allow-empty -m "Smoke commit subject"
printf '{"version":1,"outcome":"Smoke outcome","source":{"kind":"smoke-test"}}' |
	"$binary" capture "$repository" HEAD >/dev/null || fail "capture"
"$binary" inspect | grep -q '"outcome": "Smoke outcome"' || fail "inspect does not show the captured outcome"

printf 'y\n' | "$binary" day >/dev/null || fail "day"
journal="$HOME/Documents/worklog/work/$(date -u +%Y)/$(date -u +%Y-%m)/$(date -u +%Y-%m-%d).md"
grep -q "Smoke outcome" "$journal" || fail "journal does not contain the captured outcome"
"$binary" status | grep -q "up to date" || fail "status does not report the written journal"

echo "smoke test passed: $("$binary" version)"
