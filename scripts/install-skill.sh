#!/usr/bin/env sh
set -eu

mode="symlink"
if [ "${1:-}" = "--copy" ]; then
  mode="copy"
elif [ "${1:-}" != "" ]; then
  echo "usage: $0 [--copy]" >&2
  exit 2
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)
source_dir="$repo_dir/skills/pco"

codex_home="${CODEX_HOME:-$HOME/.codex}"
target_root="$codex_home/skills"
target_dir="$target_root/pco"

if [ ! -f "$source_dir/SKILL.md" ]; then
  echo "missing skill source: $source_dir/SKILL.md" >&2
  exit 1
fi

mkdir -p "$target_root"

if [ -e "$target_dir" ] || [ -L "$target_dir" ]; then
  if [ -L "$target_dir" ]; then
    rm "$target_dir"
  else
    backup="$target_dir.backup.$(date +%Y%m%d%H%M%S)"
    mv "$target_dir" "$backup"
    echo "moved existing skill to $backup"
  fi
fi

if [ "$mode" = "copy" ]; then
  cp -R "$source_dir" "$target_dir"
  echo "installed pco skill copy at $target_dir"
else
  ln -s "$source_dir" "$target_dir"
  echo "installed pco skill symlink at $target_dir"
  echo "future git pulls in $repo_dir will update the installed skill"
fi

