#!/bin/sh
# preamble.sh [portraits|objects|board|icons]
# Prints the art/style.md "## Preamble" text block, a blank line, and the text block of the named
# section (default portraits).
set -eu
kind=${1:-portraits}
here=$(cd "$(dirname "$0")" && pwd)
repo=$(cd "$here/../../../.." && pwd)
{ [ -f "$repo/CLAUDE.md" ] && [ -f "$repo/go.mod" ]; } || repo=/mnt/projekte/Code/RuderspielVera
style=$repo/art/style.md
case $kind in
portraits) sec=Portraits ;; objects) sec=Objects ;; board) sec=Board ;; icons) sec=Icons ;;
*)
	echo "unknown kind $kind (portraits|objects|board|icons)" >&2
	exit 2
	;;
esac
block() { # first ```text block under "## $1"
	awk -v h="## $1" '
    $0 == h { in_sec = 1; next }
    in_sec && /^## / { exit }
    in_sec && !inb && $0 == "```text" { inb = 1; next }
    inb && $0 == "```" { exit }
    inb { print }' "$style"
}
out=$(block Preamble)
[ -n "$out" ] || {
	echo "no Preamble block in $style" >&2
	exit 1
}
sect=$(block "$sec")
[ -n "$sect" ] || {
	echo "no $sec block in $style" >&2
	exit 1
}
printf '%s\n\n%s\n' "$out" "$sect"
