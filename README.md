# pco-cli

Command-line tools for Planning Center workflows.

## Build and Test

```sh
make test
make build
```

Install the CLI into `GOBIN`:

```sh
make install
```

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

## Codex Skill

This repo includes a companion Codex skill at `skills/pco/SKILL.md`. The skill teaches agents how to use this CLI safely, including when to inspect Planning Center state before making mutations.

Install or refresh the skill:

```sh
make install-skill
```

By default, the installer symlinks `skills/pco` into `$CODEX_HOME/skills/pco`, or `~/.codex/skills/pco` when `CODEX_HOME` is unset. Because it is a symlink, pulling updates in this repo keeps the installed skill current.

To install by copying instead of symlinking:

```sh
scripts/install-skill.sh --copy
```

