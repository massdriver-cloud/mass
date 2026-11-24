# Get Event Destination

Shows an event destination: its status, the events it selects, and its config. The API never returns the destination's `auth`.

`-o json` includes the destination's `signingSecret`.

## Usage

```bash
mass event destination get <id> [flags]
```

## Flags

- `--output, -o`: Output format, `text` (default) or `json`

## Examples

```bash
mass event destination get 4f9b2c1e-8d3a-4c6e-9f1b-2a7d5e8c0b13

# Read the signing secret again
mass event destination get 4f9b2c1e-8d3a-4c6e-9f1b-2a7d5e8c0b13 -o json | jq -r .signingSecret
```
