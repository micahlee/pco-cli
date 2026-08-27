# pco-cli

Command-line tools for Planning Center workflows.

## Build and Test

```sh
make test
make build
```

## Plan Export

Export a normalized Planning Center service plan document for comparison tools:

```sh
pco plans export <plan-id> --json
```

The export includes plan metadata, ordered items, item descriptions and HTML details, item notes, header context, song IDs/titles, arrangement IDs/names, and linked media IDs. Add `--include-raw` when troubleshooting Planning Center API behavior:

```sh
pco plans export <plan-id> --json --include-raw
```

Inspect a song's arrangements, including tempo and meter for worship planning:

```sh
pco songs arrangements <song-id> --json
```

## Safe Song Workflows

### Replace a song in one service plan

For routine changes, identify the plan by date and use the current title as a stale-state guard:

```sh
pco songs replace \
  --date 2026-09-13 \
  --current "Ancient of Days" \
  --with "I Am Not My Own" \
  --dry-run --json

pco songs replace \
  --date 2026-09-13 \
  --current "Ancient of Days" \
  --with "I Am Not My Own" \
  --json
```

You can use `--plan-id <id>` instead of `--date`; when both are supplied, the plan date must agree. Replacement fails without mutation unless it resolves exactly one plan, current item, replacement library song, and arrangement. Immediately before changing the item, the CLI rereads it and confirms that the expected current song is still assigned. It rereads again afterward and verifies the replacement relationship.

Arrangement selection is deliberately conservative:

- Use the unique active arrangement named `Default Arrangement`.
- Otherwise use the sole active arrangement.
- Otherwise require `--arrangement-id <id>`.
- Use `--song-only` only when an intentionally blank arrangement is appropriate.

`--dry-run` returns a `preview`; a repeated completed request returns an idempotent `noop`; a verified mutation returns `success`. `--json` uses a stable envelope with `schema_version`, `operation`, `status`, mutation/verification flags, resolved resources, warnings, and structured errors.

The older commands remain available as low-level recovery tools:

```sh
pco songs set <plan-id> <item-id> <song-id> [--arrangement-id <id> | --song-only]
pco songs add <plan-id> <after-item-id> <song-id> [--arrangement-id <id> | --song-only]
```

### Create a library song and arrangement

Creation is explicit and never places the new song in a plan:

```sh
pco songs create \
  --title "I Am Not My Own" \
  --authors "Keith Getty, Kristyn Getty, Matt Boswell, Matt Papa, Skye Peterson" \
  --arrangement-name "Default Arrangement" \
  --ccli 7217781 \
  --key D \
  --bpm 72 \
  --meter 4/4 \
  --json
```

Use `--no-ccli` instead of `--ccli` only when the song truly has no CCLI number. Key, BPM, and meter may be omitted, but the result includes visible warnings for every missing value.

Before creating anything, the CLI searches by CCLI (when supplied) and normalized exact title. It reports plausible matches and stops; it never silently reuses them. `--allow-duplicate` is an explicit override to create a separate record.

Planning Center exposes song and arrangement creation as separate resources. If the song is created but arrangement creation or verification fails, the CLI does not delete the song. Its structured partial result lists exactly what was created and supplies a resumable command using `--resume-song-id <id>`.

The CLI handles record resolution and mutation only. Theological judgment, musical selection, service-flow policy, and repetition policy belong in the calling workflow, not in `pco`.

Install the CLI into `GOBIN`:

```sh
make install
```

For release downloads and agent-tool setup, see [docs/install.md](docs/install.md).

## Configuration

Run interactive setup on macOS:

```sh
pco init
```

This stores `PCO_CLIENT_ID` and `PCO_SECRET` in the macOS Keychain.

You can also configure credentials with environment variables:

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

Or use `~/.config/pco/config.yaml`:

```yaml
client_id: your_client_id
client_secret: your_secret
person_id: "20101843"
service_type_id: "643436"
band_team_id: "2461416"
service_resp_team_id: "2839232"
default_template_id: "50925693"
```

## Agent Skill

This repo includes a companion agent skill at `.agents/skills/pco/SKILL.md`. The skill teaches AI agent tools how to use this CLI safely, including when to inspect Planning Center state before making mutations.

Install or refresh for Codex:

```sh
make install-skill-codex
```

Install or refresh for Claude Code:

```sh
make install-skill-claude
```

Install the Cursor rule adapter into a project:

```sh
make install-cursor-rule CURSOR_PROJECT_DIR=/path/to/project
```

By default, the installer symlinks the skill/rule into the target tool. Because it is a symlink, pulling updates in this repo keeps the installed skill current.

To install by copying instead of symlinking:

```sh
scripts/install-skill.sh --tool codex --copy
scripts/install-skill.sh --tool claude --copy
scripts/install-skill.sh --tool cursor --copy --project-dir /path/to/project
```
