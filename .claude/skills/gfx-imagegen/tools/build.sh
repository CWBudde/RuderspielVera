#!/bin/sh
# build.sh: builds every helper in this folder to ../bin/<name> (git-ignored).
set -eu
here=$(cd "$(dirname "$0")" && pwd)
mkdir -p "$here/../bin"
for d in "$here"/*/; do
	name=$(basename "$d")
	[ -f "$d/go.mod" ] || continue
	(cd "$d" && go build -buildvcs=false -o "$here/../bin/$name" .)
	echo "built bin/$name"
done
