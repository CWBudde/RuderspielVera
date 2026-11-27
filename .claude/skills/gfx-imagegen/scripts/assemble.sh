#!/bin/sh
# assemble.sh <kind> <entry> "<attachment line>" > prompt.txt
#
# <kind>  portraits|objects|board|icons: art/style.md Preamble + section text, a blank line, the
#         entry, a blank line, the attachment line ("" when nothing is attached).
# <entry> an asset ID from art/prompts.md (the first ```text block after the line starting
#         "**<ID> " or "**<ID>**"), or a file holding the entry text (an edited or extended entry).
set -eu
[ $# -eq 3 ] || { echo 'usage: assemble.sh <kind> <entry-ID|file> "<attachment line>"' >&2; exit 2; }
kind=$1; entry=$2; attach=$3
here=$(cd "$(dirname "$0")" && pwd)
repo=$(cd "$here/../../../.." && pwd)
{ [ -f "$repo/CLAUDE.md" ] && [ -f "$repo/go.mod" ]; } || repo=/mnt/projekte/Code/RuderspielVera
if [ -f "$entry" ]; then
  text=$(cat "$entry")
else
  text=$(awk -v id="$entry" '
    !found && (index($0, "**" id " ") == 1 || index($0, "**" id "**") == 1) { found = 1; next }
    found && !inb && $0 == "```text" { inb = 1; next }
    inb && $0 == "```" { exit }
    inb { print }' "$repo/art/prompts.md")
  [ -n "$text" ] || { echo "no entry $entry in art/prompts.md" >&2; exit 1; }
fi
"$here/preamble.sh" "$kind"
echo
if [ -n "$attach" ]; then
  printf '%s\n\n%s\n' "$text" "$attach"
else
  printf '%s\n' "$text"
fi
