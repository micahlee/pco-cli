#!/usr/bin/env sh
set -eu

version="${1:-dev}"
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)
skill_dir="$repo_dir/.agents/skills/pco"
dist_dir="$repo_dir/dist"
work_dir="$dist_dir/skill-package"
archive="$dist_dir/pco-agent-skill_${version}.zip"

if ! command -v zip >/dev/null 2>&1; then
  echo "zip is required to package the agent skill" >&2
  exit 1
fi

if [ ! -f "$skill_dir/SKILL.md" ]; then
  echo "missing skill source: $skill_dir/SKILL.md" >&2
  exit 1
fi

rm -rf "$work_dir" "$archive"
mkdir -p "$work_dir"
cp -R "$skill_dir" "$work_dir/pco"

(cd "$work_dir" && zip -qr "$archive" pco)
rm -rf "$work_dir"

echo "wrote $archive"
