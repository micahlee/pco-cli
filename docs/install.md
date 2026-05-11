# Install pco-cli and Agent Skill

This repo ships two things:

- `pco`, the Planning Center CLI
- `pco`, the companion agent skill/rule for tools such as Codex, Claude Code, and Cursor

Install the CLI and the skill separately. The skill does not vendor the CLI binary; it teaches AI tools how to use the installed `pco` command safely.

## From Source

```sh
git clone https://github.com/micahlee/pco-cli.git
cd pco-cli
make install
```

Update from source:

```sh
git pull --ff-only
make install
```

## GitHub Releases

Download the archive for your platform from the latest release, then put `pco` somewhere on your `PATH`.

Release archives are named like:

```text
pco_0.1.0_darwin_arm64.tar.gz
pco_0.1.0_linux_amd64.tar.gz
pco_0.1.0_windows_amd64.zip
pco-agent-skill_0.1.0.zip
checksums.txt
```

Verify checksums before installing release artifacts when possible.

## Configure Credentials

On macOS:

```sh
pco init
```

Or use environment variables:

```sh
export PCO_CLIENT_ID=your_client_id
export PCO_SECRET=your_secret
```

Optional environment variables:

```sh
export PCO_PERSON_ID=20101843
export PCO_SERVICE_TYPE_ID=643436
export PCO_BAND_TEAM_ID=2461416
export PCO_SERVICE_RESP_TEAM_ID=2839232
export PCO_DEFAULT_TEMPLATE_ID=50925693
```

## Install the Agent Skill

The canonical skill lives at:

```text
.agents/skills/pco/SKILL.md
```

Install for Codex:

```sh
make install-skill-codex
```

Install for Claude Code:

```sh
make install-skill-claude
```

Install a Cursor project rule into the current project:

```sh
make install-cursor-rule CURSOR_PROJECT_DIR=/path/to/project
```

By default, Codex and Claude installs use symlinks. This is best for development because `git pull` updates the installed skill.

To copy instead:

```sh
scripts/install-skill.sh --tool codex --copy
scripts/install-skill.sh --tool claude --copy
scripts/install-skill.sh --tool cursor --copy --project-dir /path/to/project
```

## Install Skill from GitHub Release Zip

Release zips contain this shape:

```text
pco/
  SKILL.md
```

Codex:

```sh
unzip pco-agent-skill_0.1.0.zip -d ~/.codex/skills/
```

Claude Code:

```sh
unzip pco-agent-skill_0.1.0.zip -d ~/.claude/skills/
```

Cursor users should install the adapter rule instead:

```sh
mkdir -p /path/to/project/.cursor/rules
cp adapters/cursor/pco.mdc /path/to/project/.cursor/rules/pco.mdc
```
