#!/bin/sh
# Remote commands are built here on purpose (local expansion; ~ expands on .20):
# shellcheck disable=SC2029,SC2087,SC2088
# gen.sh <id> <prompt.txt> [ref.png ...]
#
# Generates ONE image with Codex's built-in image tool on the graphics machine 192.168.213.20 over ssh
# (its own Codex account). NEVER runs a local codex: this machine's codex is a different account.
#
# <id> is <group>-<NN>, e.g. portraits-01; <group> is the id up to its last "-". The prompt and refs go to
# art/gen/$GEN_DIR/<group>/ locally and under the same relative path in the dedicated remote work dir
# /mnt/Projekte/Code/games/RuderspielVera on .20 (the only place written there; never delete files there).
# Back come <id>.png, <id>.jsonl, <id>-last.txt.
# The local <id>.png is the run's own image, fetched by its exec id from ~/.codex/generated_images/ on
# .20 (Codex's copy to <id>.png on .20 never overwrites, so that file can hold an older try).
# Refuses an <id> whose <id>.png already exists on .20 or here, unless GEN_FORCE=1.
# Last line: "OK <local png> <exec-....png or thread-<id> when Codex wrote the file itself>", or "FAILED <id>..." with the log tail before it (exit 1).
# Warnings start with "WARN".
#
# Env: GEN_DIR   date folder under art/gen/ (default YYYY-MM-DD, today)
#      GEN_FORCE 1 = run even if <id>.png exists (the new image replaces the local one; .20 keeps its file)
#      GEN_MAX   concurrent Codex runs allowed on .20 (default 3; never raise it)
#      GEN_WAIT  seconds to wait for a free slot before giving up (default 1800)
# Takes 1-3 minutes per image. Runnable from any cwd.
set -eu
[ $# -ge 2 ] || { echo "usage: gen.sh <id> <prompt.txt> [ref.png ...]" >&2; exit 2; }
id=$1; prompt=$2; shift 2
case $id in *-*) ;; *) echo "id must be <group>-<NN>" >&2; exit 2;; esac
[ -f "$prompt" ] || { echo "no prompt file $prompt" >&2; exit 2; }
for r in "$@"; do [ -f "$r" ] || { echo "no ref file $r" >&2; exit 2; }; done

# Hard-coded on purpose: the gfx machine, the dedicated Ruderspiel work dir there and its codex binary.
host=192.168.213.20
remote=/mnt/Projekte/Code/games/RuderspielVera
codex='~/.local/bin/codex'

here=$(cd "$(dirname "$0")" && pwd)
repo=$(cd "$here/../../../.." && pwd)   # .claude/skills/gfx-imagegen/scripts -> repo root
{ [ -f "$repo/CLAUDE.md" ] && [ -f "$repo/go.mod" ]; } || repo=/mnt/projekte/Code/RuderspielVera
gendir=${GEN_DIR:-$(date +%F)}
case $gendir in */*|.*|"") echo "GEN_DIR must be a plain folder name such as 2026-11-27" >&2; exit 2;; esac
group=${id%-*}
rel=art/gen/$gendir/$group
local=$repo/$rel
max=${GEN_MAX:-3}; [ "$max" -le 3 ] || max=3
wait=${GEN_WAIT:-1800}
force=${GEN_FORCE:-0}

# An existing <id>.png means a retry under the same id: Codex would not overwrite it on .20.
listing=$(ssh $host "ls -1 $remote/$rel 2>/dev/null || true")
has() { [ -e "$local/$1.png" ] || printf '%s\n' "$listing" | grep -qxF "$1.png"; }
old_sha=
if has "$id"; then
  where=; [ -e "$local/$id.png" ] && where="here"
  printf '%s\n' "$listing" | grep -qxF "$id.png" && where="${where:+$where and }on $host"
  if [ "$force" != 1 ]; then
    alt=
    for s in b c d e f g h i j k l m n o p q r s t u v w x y z; do has "$id$s" || { alt=$id$s; break; }; done
    echo "$id.png already exists $where; use a new id such as ${alt:-<id>b}, or GEN_FORCE=1 to run anyway" >&2
    echo "FAILED $id: $id.png exists $where (try ${alt:-a new id} or GEN_FORCE=1)"
    exit 1
  fi
  old_sha=$(ssh $host "sha256sum $remote/$rel/$id.png 2>/dev/null || true" | cut -d' ' -f1)
  echo "WARN $id.png exists $where; GEN_FORCE=1: the new image replaces the local one, .20 keeps its file"
fi

# At most $max Codex runs at once on .20 (other agents share it). Count real codex processes only.
waited=0
while :; do
  n=$(ssh $host "pgrep -c -f '^/home/christian/[.]local/bin/codex exec' || true")
  [ "${n:-0}" -lt "$max" ] && break
  [ $waited -ge "$wait" ] && { echo "FAILED $id: $n Codex runs busy on $host for ${waited}s"; exit 1; }
  sleep 20; waited=$((waited + 20))
done
free=$(ssh $host "df -Pk $remote | awk 'NR==2{print \$4}'")
[ "${free:-0}" -gt 2097152 ] || { echo "FAILED $id: under 2 GB free on $host:$remote" >&2; exit 1; }
# Codex writes its transcript and generated images under ~/.codex (the root disk on .20); when that is
# full, the image tool "loses" its result and Codex reports a missing PNG.
hfree=$(ssh $host "df -Pk ~/.codex | awk 'NR==2{print \$4}'")
[ "${hfree:-0}" -gt 1048576 ] || { echo "FAILED $id: under 1 GB free for ~/.codex on $host (root disk full); free space there first" >&2; echo "FAILED $id: ~/.codex disk full on $host"; exit 1; }

mkdir -p "$local"
ssh $host "mkdir -p $remote/$rel"   # only under the dedicated work dir
[ "$(readlink -f "$prompt")" = "$(readlink -f "$local/$id-prompt.txt")" ] || cp "$prompt" "$local/$id-prompt.txt"
scp -q "$local/$id-prompt.txt" "$host:$remote/$rel/$id-prompt.txt"
imgs=""
for r in "$@"; do
  b=$(basename "$r")
  scp -q "$r" "$host:$remote/$rel/$b"
  imgs="$imgs -i $rel/$b"
done
msg="Use the imagegen skill with the built-in image_gen tool. Generate exactly one image from the prompt in the stdin block, with the attached images as references as the prompt describes. Copy the generated PNG unchanged to $rel/$id.png. Never copy any other PNG: other agents generate images at the same time, so if your own image was not saved, do not search ~/.codex/generated_images for the newest file; say that the image is missing instead. Do not create, edit or delete any other file, do not run other commands, do not commit."
ssh $host "cd $remote && $codex exec --skip-git-repo-check -C $remote -s workspace-write --json -o $rel/$id-last.txt $imgs \"$msg\" < $rel/$id-prompt.txt > $rel/$id.jsonl 2>&1" || true
scp -q "$host:$remote/$rel/$id.jsonl" "$local/" 2>/dev/null || true
scp -q "$host:$remote/$rel/$id-last.txt" "$local/" 2>/dev/null || true

fail() {
  echo "FAILED $id${1:+: $1}"; tail -c 800 "$local/$id.jsonl" 2>/dev/null || true
  cat "$local/$id-last.txt" 2>/dev/null || true
  exit 1
}

# The run's own image lives in ~/.codex/generated_images/<thread>/exec-<uuid>.png on .20, where
# <thread> is the thread_id of this run's log. Only that folder counts: when the image tool did not
# save a file, Codex lists the whole generated_images/ folder while searching, so exec names in the log
# can belong to other (concurrent) runs. Take the newest non-empty file (Codex has left 0-byte exec
# files behind); without one, fall back to the <id>.png Codex wrote (it then streams the image itself),
# unless that file equals an exec image of another thread from the last two days: Codex then grabbed
# the newest PNG of some other (concurrent) run, which has happened.
thread=$(grep -o '"thread_id":"[0-9a-f-]*"' "$local/$id.jsonl" 2>/dev/null | head -1 | cut -d'"' -f4 || true)
ex=
info=$(ssh $host 'sh -s' <<EOF
p=
if [ -n "$thread" ]; then
  for f in \$(ls -1t \$HOME/.codex/generated_images/$thread/exec-*.png 2>/dev/null); do [ -s "\$f" ] && { p=\$f; break; }; done
fi
[ -n "\$p" ] && echo "EXEC \$p \$(sha256sum < "\$p" | cut -d' ' -f1)"
if [ -s "$remote/$rel/$id.png" ]; then
  s=\$(sha256sum < "$remote/$rel/$id.png" | cut -d" " -f1)
  echo "ID \$s"
  if [ -z "\$p" ]; then
    for f in \$(find \$HOME/.codex/generated_images -name "exec-*.png" -size +0 -mtime -2 2>/dev/null); do
      [ "\$(sha256sum < "\$f" | cut -d" " -f1)" = "\$s" ] && { echo "FOREIGN \$f"; break; }
    done
  fi
fi
true
EOF
) || true
exec_path=$(printf '%s\n' "$info" | awk '$1=="EXEC"{print $2}')
exec_sha=$(printf '%s\n' "$info" | awk '$1=="EXEC"{print $3}')
id_sha=$(printf '%s\n' "$info" | awk '$1=="ID"{print $2}')
foreign=$(printf '%s\n' "$info" | awk '$1=="FOREIGN"{print $2}')

tmp=$local/.$id.png.part
if [ -n "$exec_path" ]; then
  ex=$(basename "$exec_path")
  if [ -n "$id_sha" ] && [ "$id_sha" != "$exec_sha" ]; then
    echo "WARN $host:$rel/$id.png differs from $ex (an older try?); using $ex"
  elif [ -z "$id_sha" ]; then
    echo "WARN Codex did not copy $id.png on $host; using $ex"
  fi
  scp -q "$host:$exec_path" "$tmp" 2>/dev/null || { rm -f "$tmp"; fail "could not fetch $exec_path"; }
  got=$(sha256sum < "$tmp" | cut -d' ' -f1)
  [ "$got" = "$exec_sha" ] || { rm -f "$tmp"; fail "fetched $ex has sha $got, expected $exec_sha"; }
elif [ -n "$id_sha" ]; then
  [ "$id_sha" != "$old_sha" ] || fail "no exec image found and $host:$rel/$id.png is the old file"
  [ -z "$foreign" ] || fail "no exec image of thread $thread; $rel/$id.png is a copy of another run's $foreign"
  echo "WARN exec image ${ex:-of thread $thread} not found on $host; using $rel/$id.png"
  scp -q "$host:$remote/$rel/$id.png" "$tmp" 2>/dev/null || { rm -f "$tmp"; fail "could not fetch $id.png"; }
else
  fail
fi
[ -s "$tmp" ] || { rm -f "$tmp"; fail "fetched image is empty"; }
mv -f "$tmp" "$local/$id.png"
echo "OK $local/$id.png ${ex:-thread-$thread}"
