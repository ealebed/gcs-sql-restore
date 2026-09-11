# ADR-005: Auto-merge Dependabot minor/patch PRs with a GitHub App

## Status
Accepted

## Date
2026-09-11

## Context
This repository had no GitHub Actions CI. Dependabot already opens daily PRs for Go modules, Terraform providers, and GitHub Actions. Manual review of minor and patch bumps is slow and low-value. We still want humans to review major (and non-semver) updates, and we need CI to exist before auto-merge is enabled.

Constraints:

- `GITHUB_TOKEN` reviews are attributed to `github-actions[bot]`. That identity is not a CODEOWNER.
- Official [CODEOWNERS](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners) syntax is users and teams only. GitHub App reviews land as `app-name[bot]` and do not count as code-owner approvals.
- This repository is personal (`ealebed/gcs-sql-restore`), so organization ruleset bypass lists are not available.
- Dependabot-triggered `pull_request` workflows receive **Dependabot secrets only**, and `GITHUB_TOKEN` is read-only by default. [Source](https://docs.github.com/en/code-security/reference/supply-chain-security/dependabot-on-actions)
- Path-filtered jobs that never report a required check block merge (or, if not required, skip the real gate).

## Decision
Add a CI workflow with two jobs that **always** run on pull requests and on `master`: `test` (Go fmt, vet, golangci-lint, race tests) and `terraform` (`terraform fmt -check`, `init -backend=false`, `validate`, TFLint). No path filters.

Use a dedicated GitHub App (`automerger`) from a GitHub Actions workflow to approve and squash-auto-merge **semver-minor** and **semver-patch** Dependabot PRs.

Keep `.github/CODEOWNERS` so humans still get review requests. Do **not** enable “Require review from Code Owners”. Require **one** approving review plus required status checks `test` and `terraform`. The auto-merge workflow runs only when the PR author is `dependabot[bot]`.

`gh pr merge --auto --squash` queues the merge. GitHub performs the squash only after required checks pass. Required checks must exist before auto-merge is useful, or a PR can merge with no CI.

Same merge pattern as `ealebed/token-injector`. Terraform fmt/validate/lint follows `ealebed/gcp-terraform-modules` and `ealebed/flyway-validation-example`. Go CI follows `ealebed/restarter`.

## Alternatives Considered

### Terraform-only CI with path filters
- Pros: Matches “at least terraform” literally.
- Cons: gomod and github-actions PRs would not report a terraform check; requiring it would block those PRs, not requiring it would auto-merge Go changes with no tests.
- Rejected: Both jobs always run.

### Fine-grained PAT of `@ealebed` (CODEOWNER)
- Pros: Approval would satisfy “Require review from Code Owners”.
- Cons: Long-lived credential tied to a person; revocation or expiry silently stops automation.
- Rejected: The App is the intended identity, and we accepted dropping the code-owner merge gate.

### `GITHUB_TOKEN` / `github-actions[bot]`
- Pros: No extra secrets.
- Cons: Does not satisfy code-owner reviews; still needs “Allow GitHub Actions to create and approve pull requests”; weaker attribution.
- Rejected: We want a dedicated App identity for approve/merge.

### Checkov in CI
- Pros: Extra static security scanning.
- Cons: High false-positive rate on GCP resources in sibling repos; not needed for a fmt/lint/validate gate.
- Rejected: TFLint + `terraform validate` are the terraform lint/validate pair.

## Consequences
- Human PRs still request `@ealebed`; they are not auto-approved.
- Major and `semver-unknown` updates stay open for manual review.
- `APP_CLIENT_ID` and `APP_PRIVATE_KEY` must exist in **both** Actions and Dependabot secret stores under identical names.
- Do not store an OAuth client secret; installation tokens need the App private key PEM.
- Client ID is read from secrets (not Actions variables) so Dependabot-triggered jobs can see it.
- Branch protection must require `test` and `terraform` after those names appear on a PR. This repo had no prior checks.
