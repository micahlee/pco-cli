---
name: pco
description: Use when the user asks to inspect or change Planning Center data through the `pco` CLI, including blockouts, plans, songs, teams, scheduling, availability checks, or music planning workflows. This is the command-level companion skill for github.com/micahlee/pco-cli; broader worship scheduling policy belongs in a separate Planning Center music/domain skill.
---

# PCO CLI

Use the `pco` command for Planning Center operations. Prefer read commands first, then mutate only after the relevant IDs, dates, people, and positions are clear.

## Setup

- Repo: `github.com/micahlee/pco-cli`
- Local repo path is often `/Users/micahlee/projects/pco-cli`.
- Install/update the skill from that repo with `make install-skill`.
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
pco plans items <plan-id>
pco plans templates
pco songs search --query "<title>"
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
pco songs set <plan-id> <item-id> <song-id>
pco songs add <plan-id> <after-item-id> <song-id> --label "<label>"
pco teams schedule <plan-id> <person-id> <team-id> "<position>"
pco teams unschedule <plan-id> <assign-id>
pco teams enable-signups <plan-id> --team-id <team-id>
```

Before mutating:

- Confirm the date and plan ID.
- Inspect current state with the relevant read command.
- For scheduling, check availability first with `pco music availability <YYYY-MM-DD>`.
- For assignment changes, inspect current team members with `pco teams show <plan-id>`.
- For song changes, inspect plan items with `pco plans items <plan-id>` and search/history as needed.
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

## Failure Modes

- Missing credentials: run `pco init` locally or set `PCO_CLIENT_ID` and `PCO_SECRET`.
- Wrong account/config: run `pco me` before making important changes.
- Ambiguous names: use IDs from `pco music team`, `pco teams show`, `pco songs search`, or `pco plans list`.
- Unexpected API errors: stop, show the command and error, and inspect current state before retrying.
