#!/usr/bin/env sh
set -eu

usage() {
  cat >&2 <<'EOF'
usage: scripts/install-skill.sh [--tool codex|claude|cursor] [--copy] [--project-dir PATH]

Defaults:
  --tool codex
  symlink install for codex/claude
  --project-dir $PWD for cursor

Examples:
  scripts/install-skill.sh --tool codex
  scripts/install-skill.sh --tool claude
  scripts/install-skill.sh --tool cursor --project-dir /path/to/project
  scripts/install-skill.sh --tool codex --copy
EOF
}

tool="codex"
mode="symlink"
project_dir="$PWD"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --tool)
      shift
      tool="${1:-}"
      ;;
    --copy)
      mode="copy"
      ;;
    --project-dir)
      shift
      project_dir="${1:-}"
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      usage
      exit 2
      ;;
  esac
  shift
done

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)
skill_dir="$repo_dir/.agents/skills/pco"
cursor_rule="$repo_dir/adapters/cursor/pco.mdc"

install_path() {
  source_path="$1"
  target_path="$2"
  mkdir -p "$(dirname -- "$target_path")"

  if [ -e "$target_path" ] || [ -L "$target_path" ]; then
    if [ -L "$target_path" ]; then
      rm "$target_path"
    else
      backup="$target_path.backup.$(date +%Y%m%d%H%M%S)"
      mv "$target_path" "$backup"
      echo "moved existing file/directory to $backup"
    fi
  fi

  if [ "$mode" = "copy" ]; then
    cp -R "$source_path" "$target_path"
    echo "installed copy at $target_path"
  else
    ln -s "$source_path" "$target_path"
    echo "installed symlink at $target_path"
    echo "future git pulls in $repo_dir will update this install"
  fi
}

case "$tool" in
  codex)
    if [ ! -f "$skill_dir/SKILL.md" ]; then
      echo "missing skill source: $skill_dir/SKILL.md" >&2
      exit 1
    fi
    codex_home="${CODEX_HOME:-$HOME/.codex}"
    install_path "$skill_dir" "$codex_home/skills/pco"
    ;;
  claude)
    if [ ! -f "$skill_dir/SKILL.md" ]; then
      echo "missing skill source: $skill_dir/SKILL.md" >&2
      exit 1
    fi
    claude_home="${CLAUDE_HOME:-$HOME/.claude}"
    install_path "$skill_dir" "$claude_home/skills/pco"
    ;;
  cursor)
    if [ ! -f "$cursor_rule" ]; then
      echo "missing Cursor rule: $cursor_rule" >&2
      exit 1
    fi
    install_path "$cursor_rule" "$project_dir/.cursor/rules/pco.mdc"
    ;;
  *)
    usage
    exit 2
    ;;
esac
