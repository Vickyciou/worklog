#!/bin/sh
# Build release executables and checksums into dist/.
# Usage: scripts/release.sh vX.Y.Z
set -eu

version=${1:-}
case "$version" in
v[0-9]*.[0-9]*.[0-9]*) ;;
*)
	echo "usage: scripts/release.sh vX.Y.Z" >&2
	exit 2
	;;
esac

root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"
commit=$(git rev-parse --short=12 HEAD)
targets="darwin/arm64 darwin/amd64 linux/amd64 linux/arm64"

rm -rf dist
mkdir dist
for target in $targets; do
	os=${target%/*}
	arch=${target#*/}
	output="dist/worklog_${version}_${os}_${arch}"
	CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath \
		-ldflags "-s -w -X main.version=$version -X main.commit=$commit" \
		-o "$output" ./cmd/worklog
done

# A release binary must not carry paths from the machine that built it.
for binary in dist/worklog_*; do
	for private in "$HOME" "$root"; do
		if LC_ALL=C grep -aqF "$private" "$binary"; then
			echo "release: $binary contains build-machine path $private" >&2
			exit 1
		fi
	done
done

count=$(ls dist/worklog_* | wc -l | tr -d ' ')
if [ "$count" -ne 4 ]; then
	echo "release: built $count executables, want 4" >&2
	exit 1
fi

cd dist
if command -v sha256sum >/dev/null 2>&1; then
	sha256sum worklog_* >SHA256SUMS
else
	shasum -a 256 worklog_* >SHA256SUMS
fi
echo "Built worklog $version ($commit):"
cat SHA256SUMS
