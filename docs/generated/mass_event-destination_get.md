---
id: mass_event-destination_get.md
slug: /cli/commands/mass_event-destination_get
title: Mass Event-Destination Get
sidebar_label: Mass Event-Destination Get
---
## mass event-destination get

Get an event destination

### Synopsis

# Get Event Destination

Shows an event destination: its status, the events it selects, and its config. The API never returns the destination's `auth`.

`-o json` includes the destination's `signingSecret`.

## Usage

```bash
mass event-destination get <id> [flags]
```

## Flags

- `--output, -o`: Output format, `text` (default) or `json`

## Examples

```bash
mass event-destination get 4f9b2c1e-8d3a-4c6e-9f1b-2a7d5e8c0b13

# Read the signing secret again
mass event-destination get 4f9b2c1e-8d3a-4c6e-9f1b-2a7d5e8c0b13 -o json | jq -r .signingSecret
```


```
mass event-destination get <id> [flags]
```

### Options

```
  -h, --help            help for get
  -o, --output string   Output format (text or json) (default "text")
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass event-destination](/cli/commands/mass_event-destination)	 - Event destination management
