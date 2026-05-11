# Security Checks

This repo uses GitHub Actions to reduce the risk of leaking secrets and shipping vulnerable dependencies.

## Gitleaks

Workflow: `.github/workflows/gitleaks.yml`

Runs on pull requests, pushes to `main`, and manual dispatch. It scans Git history and the proposed diff for likely secrets.

Local check:

```sh
gitleaks detect --source . --verbose
```

## Snyk

Workflow: `.github/workflows/snyk.yml`

Runs on pull requests, pushes to `main`, and manual dispatch. It scans Go dependencies when `go.mod` exists.

Required repository secret:

```text
SNYK_TOKEN
```

The Snyk job is guarded so this checks-only PR can pass before the Go module lands. Once the CLI implementation PR is based on a `main` branch containing this workflow, the Snyk job will run against `go.mod` and fail if the token is missing or vulnerabilities at medium severity or higher are found.

Local check:

```sh
snyk test --severity-threshold=medium
```

