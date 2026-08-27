---
name: pco
description: Use when the user asks to inspect or change Planning Center data through the `pco` CLI, including blockouts, plans, songs, teams, scheduling, availability checks, or music planning workflows. This is the command-level companion skill for github.com/micahlee/pco-cli; broader worship scheduling policy belongs in a separate Planning Center music/domain skill.
---

# PCO CLI

Use the `pco` command for Planning Center operations. Prefer intent-level commands for routine work; use low-level ID-based commands for diagnosis, unusual recovery, and cases the intent workflow deliberately refuses.

## Setup

- Repo: `github.com/micahlee/pco-cli`
- Local repo path is often `/Users/micahlee/projects/pco-cli`.
- Install/update the skill from that repo with `make install-skill-codex` or `make install-skill-claude`.
- Build/test the CLI with `make build` and `make test`.
- Credentials come from macOS Keychain via `pco init`, environment variables, or `~/.config/pco/config.yaml`.

Environment variables:

- `PCO_CLIENT_ID`
- `PCO_SECRET`
- `PCO_PERSON_ID`
- `PCO_SERVICE_TYPE_ID`
- `PCO_BAND_TEAM_ID`
- `PCO_SERVICE_RESP_TEAM_ID`
- `PCO_DEFAULT_TEMPLATE_ID`

## Output

Use `--json` when scripting, diffing, or feeding output into another tool. Use table output when reporting directly to the user.

## Read Commands

Use these freely to gather context:

```sh
pco me
pco blockouts list
pco plans list --count 10
pco plans show <plan-id>
pco plans export <plan-id> --json
pco plans items <plan-id>
pco plans templates
pco songs search --query "<title>"
pco songs arrangements <song-id> --json
pco songs history --weeks 16
pco teams show <plan-id>
pco music team
pco music availability <YYYY-MM-DD>
pco music month <YYYY-MM>
```

## Mutation Commands

Treat these as state-changing operations:

```sh
pco blockouts add <start-date> <end-date> --reason "<reason>"
pco blockouts delete <id>
pco plans create <YYYY-MM-DD> --template <template-id>
pco songs replace --date <YYYY-MM-DD> --current "<current title>" --with "<replacement title>"
pco songs create --title "<title>" --authors "<authors>" --arrangement-name "<name>" --ccli <number>
pco songs set <plan-id> <item-id> <song-id>
pco songs add <plan-id> <after-item-id> <song-id> --label "<label>"
pco teams schedule <plan-id> <person-id> <team-id> "<position>"
pco teams unschedule <plan-id> <assign-id>
pco teams enable-signups <plan-id> --team-id <team-id>
```

### Routine song replacement

Use `songs replace` instead of manually resolving IDs and calling `songs set`:

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

The command requires exactly one plan, current item, replacement library song, and safe arrangement choice. It rechecks the current item as a stale-state precondition and verifies after mutation. Use `--plan-id` when date resolution is ambiguous. Use `--arrangement-id` when multiple active arrangements have no unique `Default Arrangement`; use `--song-only` only when leaving the arrangement blank is intentional. Treat `preview`, `noop`, and `success` as complete structured outcomes. On an error, do not fall through to an unguarded mutation automatically—inspect the reported ambiguity or stale state first.

### Explicit library-song creation

`songs create` creates a library song and its initial arrangement, but never adds it to a plan. Require title, authors, arrangement name, and either `--ccli` or explicit `--no-ccli`:

```sh
pco songs create \
  --title "I Am Not My Own" \
  --authors "Keith Getty, Kristyn Getty, Matt Boswell, Matt Papa, Skye Peterson" \
  --arrangement-name "Default Arrangement" \
  --ccli 7217781 \
  --key D --bpm 72 --meter 4/4 \
  --json
```

The command searches both CCLI and normalized exact title. If it reports plausible duplicates, stop and inspect them. Never interpret the result as permission to reuse a match, and use `--allow-duplicate` only when the user explicitly intends a second record. Missing key, BPM, or meter is allowed but produces visible warnings.

Song and arrangement creation are two API writes. If arrangement creation or verification fails after the song exists, do not delete the song. Report the exact `created_resources` and use the returned `resume_command`, which includes `--resume-song-id`, to safely complete or verify the arrangement.

### Low-level recovery commands

Use these explicit-ID escape hatches when diagnosing API state or recovering from a refused intent workflow:

```sh
pco songs set <plan-id> <item-id> <song-id> [--arrangement-id <id> | --song-only]
pco songs add <plan-id> <after-item-id> <song-id> [--arrangement-id <id> | --song-only]
```

Do not prefer them for ordinary replacement because they bypass title/date intent resolution and the current-title stale-state guard.

Before mutating:

- Confirm the date and plan ID.
- Inspect current state with the relevant read command.
- For scheduling, check availability first with `pco music availability <YYYY-MM-DD>`.
- For assignment changes, inspect current team members with `pco teams show <plan-id>`.
- For routine song replacement, use `pco songs replace --dry-run --json`; it performs the relevant preflight itself.
- For low-level song recovery, inspect plan items and arrangements explicitly before mutation.
- If the user did not explicitly ask for the exact mutation, propose the command and wait.

After mutating:

- Re-run the relevant read command to verify the change.
- Report the command used and the resulting ID/name/status.

## Scheduling Guardrails

- `pco teams schedule` queues a notification but does not send it.
- Do not schedule someone who is blocked out unless the user explicitly overrides that.
- Do not invent positions, team IDs, person IDs, plan IDs, item IDs, or song IDs. Look them up.
- Use the configured Band team and Service Responsibilities team IDs when applicable.
- Music Lead is in the Service Responsibilities team.
- Use the broader Planning Center music/domain skill for policy questions such as rotation fairness, role preferences, and month planning.

## Scope Boundary

The CLI resolves records and carries out explicit Planning Center changes. Keep theological judgment, song selection, musical suitability, service-flow policy, and repetition policy outside the CLI. Obtain those decisions from the user or an appropriate domain workflow, then pass the chosen titles and metadata to `pco`.

## Failure Modes

- Missing credentials: run `pco init` locally or set `PCO_CLIENT_ID` and `PCO_SECRET`.
- Wrong account/config: run `pco me` before making important changes.
- Ambiguous names: use IDs from `pco music team`, `pco teams show`, `pco songs search`, or `pco plans list`.
- Unexpected API errors: stop, show the command and error, and inspect current state before retrying.
