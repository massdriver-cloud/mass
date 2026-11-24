---
id: mass_event-destination_update.md
slug: /cli/commands/mass_event-destination_update
title: Mass Event-Destination Update
sidebar_label: Mass Event-Destination Update
---
## mass event-destination update

Update an event destination

### Synopsis

# Update Event Destination

Changes an event destination. Only the flags you set change.

## Usage

```bash
mass event-destination update <id> [flags]
```

## Flags

- `--name`: New name of the destination.
- `--url`: New URL. Changes `url` in the current config.
- `--header`: A header sent with each event, `Name=value`. Adds to or changes `headers` in the current config. Repeat for more. To remove a header, use `--config`.
- `--config`: Replaces the config, as JSON or `@file`.
- `--auth`: Replaces the credentials, as JSON or `@file`. `--auth '{}'` removes them.
- `--event-type`: Replaces the event types. Repeat for more. `--event-type ''` receives every event.
- `--attribute`: Replaces the attributes, `key=value`. Repeat for more. `--attribute ''` receives events of any attributes.
- `--condition`: Replaces the jq condition. `--condition ''` removes it.

New event types and attributes apply to events recorded after the change. A new condition also applies to events that wait to be sent.

## Examples

```bash
# Rotate the credentials
mass event-destination update 4f9b2c1e-8d3a-4c6e-9f1b-2a7d5e8c0b13 --auth @auth.json

# Receive every event
mass event-destination update 4f9b2c1e-8d3a-4c6e-9f1b-2a7d5e8c0b13 --event-type ''
```


```
mass event-destination update <id> [flags]
```

### Options

```
      --attribute stringArray    Attribute the event's entity must have had, key=value; repeat for more
      --auth string              Adapter credentials as JSON, or @file. Never returned by the API
      --condition string         jq expression that must output true for the event to be sent
      --config string            Adapter config as JSON, or @file
      --event-type stringArray   Event type, or <entity>.* for every event of an entity; repeat for more
      --header stringArray       Header sent with each event, Name=value (sets headers in the HTTP config, which the API returns; put credentials in --auth); repeat for more
  -h, --help                     help for update
      --name string              Name of the destination, unique in the organization
      --url string               URL to post events to (sets url in the HTTP config)
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass event-destination](/cli/commands/mass_event-destination)	 - Event destination management
