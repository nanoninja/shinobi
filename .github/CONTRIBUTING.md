# Contributing to Shinobi

Thank you for your interest in contributing to Shinobi.

## Reporting Issues

Found a bug or have a suggestion?

1. Check if the issue already exists in [Issues](https://github.com/nanoninja/shinobi/issues).
2. If not, [open a new one](https://github.com/nanoninja/shinobi/issues/new). Please include:
   - A clear description of the problem or suggestion.
   - A minimal code example reproducing the issue.
   - Your Go version (`go version`).

## Security

To report a security vulnerability, please use [GitHub Security Advisories](https://github.com/nanoninja/shinobi/security/advisories/new) instead of opening a public issue.

## Before You Start

Open an issue to discuss your idea before submitting a pull request. This avoids duplicate work and ensures the change aligns with the project direction.

## Ground Rules

- No external dependencies. Shinobi is stdlib-only by design (except `github.com/nanoninja/render`).
- No breaking changes without a minor version bump (while at `v0.x`).
- All exported symbols must have godoc comments.
- All new code must be covered by tests. Run with `-race` to check for data races.

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <short description>

Types : feat, fix, refactor, test, docs, ci, chore
Scope : middleware, router, app, ctx, bind — or omit for top-level changes

Examples:
  feat(middleware): add RateLimit middleware
  fix(app): normalize trailing slash in wrapPath
  docs: update README with Mount example
```

## Branch Naming

```
feat/<name>      new feature
fix/<name>       bug fix
refactor/<name>  refactoring without behaviour change
```

## Pull Request Process

1. Fork the repository and create a branch following the naming convention above.
2. Make your changes with tests.
3. Run `go test -v -race ./...` and `golangci-lint run ./...` — both must pass.
4. Update `CHANGELOG.md` under `[Unreleased]`.
5. Open a pull request against `main`.

## Release Process (maintainers)

1. Move `[Unreleased]` entries to a new versioned section in `CHANGELOG.md`.
2. Commit: `git commit -m "chore: release vX.Y.Z"`.
3. Tag: `git tag vX.Y.Z && git push origin vX.Y.Z`.
4. Create a GitHub release using the CHANGELOG section as release notes.
