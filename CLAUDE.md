# CLAUDE.md

## Layering

`cmd/` is Cobra wiring only — command definitions, flag reading, client init. **There are no test files in `cmd/`; don't add any.** Logic worth testing goes in `internal/commands/<domain>` as a `RunX(...)` function, with dependencies injected so tests can fake them: a narrow `API` interface plus `NewAPI(client)`, `io.Reader` for confirmation prompts, `io.Writer` for output. Wanting to unit-test a `cmd/` helper means the logic is in the wrong package.

## The SDK owns auth and config

`massdriver/config` in the SDK is the sole authority on the config file — its schema, location, version, and which profile is active. Never resolve credentials or profiles in the CLI, and never add config keys the SDK doesn't read. If the CLI needs behavior the SDK lacks, change the SDK. The CLI only writes the file (`internal/configfile`) and passes `--profile` through as `massdriver.WithProfile`.

## Before finishing

- `make check` — tests and lint.
- `make docs` whenever a command, flag, or helpdoc changes. CI regenerates and fails on any diff.

## Comments

Default to none. Write one only when a reader who already understands the code still can't see *why*. Never narrate what the code says, and never narrate what changed or used to be there — git holds the history. No doc comments on unexported helpers.

## Tests

Table-driven, stdlib `testing`, hand-rolled fakes. One `_test.go` per source file — append cases rather than adding a file per scenario. For unexported access, switch the file to `package foo //nolint:testpackage // reason` instead of splitting the tests in two.
