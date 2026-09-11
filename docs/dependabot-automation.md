# Dependabot automation

Minor and patch Dependabot pull requests are approved and squash-merged by the `automerger` GitHub App after required CI passes. Major and non-semver updates stay open for manual review.

See [ADR-005](decisions/ADR-005-dependabot-auto-merge.md) for why a GitHub App is used and why code-owner reviews are **not** a merge gate.

## How it works

```mermaid
sequenceDiagram
  participant Dependabot
  participant Actions as GitHubActions
  participant App as AutomergerApp
  participant GH as GitHubMergeGate

  Dependabot->>GH: Open minor or patch PR
  GH->>Actions: pull_request event
  Actions->>Actions: CI test and terraform jobs
  Actions->>Actions: Dependabot auto-merge workflow
  Actions->>Actions: fetch-metadata update-type
  alt semver-minor or semver-patch
    Actions->>App: Mint installation token
    Actions->>GH: gh pr review --approve
    Actions->>GH: gh pr merge --auto --squash
    GH->>GH: Wait for required checks
    GH->>GH: Squash merge to master
  else major or unknown
    Actions->>Actions: Skip approve and merge
  end
```

What gets auto-merged:

- `version-update:semver-minor`
- `version-update:semver-patch`

What stays manual:

- `version-update:semver-major`
- `version-update:semver-unknown`
- Any PR not authored by `dependabot[bot]`

`--auto` does **not** wait inside the job. GitHub merges later, only if branch protection is satisfied. If required status checks are missing, GitHub can squash-merge as soon as the App approves.

## Repository files

- [`.github/workflows/ci.yml`](../.github/workflows/ci.yml) — Go `test` plus Terraform fmt / validate / TFLint (`terraform` job). Both jobs always run.
- [`.github/workflows/dependabot-auto-merge.yml`](../.github/workflows/dependabot-auto-merge.yml) — approve + enable squash auto-merge
- [`.github/dependabot.yml`](../.github/dependabot.yml) — daily gomod, terraform, and github-actions updates, `dependencies` label, assignee `@ealebed`
- [`.github/CODEOWNERS`](../.github/CODEOWNERS) — review requests to `@ealebed` (not a merge requirement)
- [`terraform/.tflint.hcl`](../terraform/.tflint.hcl) — TFLint terraform recommended preset + google plugin

The auto-merge workflow never checks out the pull request branch.

## GitHub App

App: `automerger` (user-owned). Webhook disabled. Installed on selected repositories.

Repository permissions:

- **Contents**: Read and write (merge)
- **Pull requests**: Read and write (approve, enable auto-merge)
- **Metadata**: Read-only (required)

The workflow mints a short-lived installation token with [`actions/create-github-app-token@v3`](https://github.com/actions/create-github-app-token) using **Client ID** + private key PEM. Do not use an OAuth client secret.

[Making authenticated API requests with a GitHub App in a workflow](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/making-authenticated-api-requests-with-a-github-app-in-a-github-actions-workflow)

## Secrets

Dependabot-triggered `pull_request` jobs only see **Dependabot** secrets, not Actions secrets or variables. Store the **same names** in both stores:

```mermaid
flowchart LR
  subgraph stores [Secret stores]
    ActionsSecrets[Actions secrets]
    DependabotSecrets[Dependabot secrets]
  end
  subgraph names [Identical names]
    ClientId[APP_CLIENT_ID]
    PrivateKey[APP_PRIVATE_KEY]
  end
  ActionsSecrets --> ClientId
  ActionsSecrets --> PrivateKey
  DependabotSecrets --> ClientId
  DependabotSecrets --> PrivateKey
  ClientId --> Workflow[dependabot-auto-merge.yml]
  PrivateKey --> Workflow
```

| Name | Store | Value |
| --- | --- | --- |
| `APP_CLIENT_ID` | Actions **and** Dependabot secrets | GitHub App Client ID (`Iv1…` / `Iv23…`) |
| `APP_PRIVATE_KEY` | Actions **and** Dependabot secrets | Full PEM, including BEGIN/END lines |

If a Dependabot run fails with an empty Client ID or private key, the values were added only under Actions secrets.

CI does **not** need GCP credentials. Terraform `validate` uses `init -backend=false`.

## Branch protection (`master`)

This repository had no CI before. After the [CI](../.github/workflows/ci.yml) workflow has reported once (this PR is enough), set:

- Require a pull request before merging
- Required approving reviews: **1**
- **Do not** require review from Code Owners
- Dismiss stale reviews when new commits are pushed (the workflow re-approves on `synchronize`)
- Require status checks to pass before merging
- Required checks (job ids; both always run, no path filters):
  - `test` — Go fmt, vet, golangci-lint, race tests
  - `terraform` — `terraform fmt -check`, `init -backend=false`, `validate`, TFLint
- Require conversation resolution: **off**
- Allow auto-merge: **on**
- Squash merging: **on**
- No force pushes, no deletions

## Rollout order

1. Open this change as a PR so `test` and `terraform` appear in Checks.
2. Add those two names as required status checks (this repo had none).
3. Confirm App install, `APP_CLIENT_ID` / `APP_PRIVATE_KEY` in both secret stores, auto-merge, and squash.
4. Merge into `master`.

Skip the required checks and a minor/patch Dependabot PR can merge before tests finish.

## Verify

1. Minor or patch Dependabot PR: App approval, auto-merge queued, squash merge after `test` and `terraform` are green.
2. Major or `semver-unknown` Dependabot PR: workflow runs, no App approval, PR stays open.
3. Human PR: auto-merge job skipped (`dependabot[bot]` guard); both CI jobs still run.
4. On a Dependabot-triggered run, `Create GitHub App token` can read both secrets.
