---
id: mass_event-destination_create.md
slug: /cli/commands/mass_event-destination_create
title: Mass Event-Destination Create
sidebar_label: Mass Event-Destination Create
---
## mass event-destination create

Create an event destination

### Synopsis

# Create Event Destination

Creates an event destination and prints its signing secret. Your receiver uses the signing secret to check the `webhook-signature` header of each request.

## Usage

```bash
mass event-destination create --name <name> [flags]
```

## Flags

- `--name`: Name of the destination, unique in the organization. Required.
- `--adapter`: How the destination receives events. Defaults to `HTTP`.
- `--url`: The URL to post events to. Sets `url` in the HTTP config.
- `--header`: A header sent with each event, `Name=value`. Sets `headers` in the HTTP config. Repeat for more.
- `--config`: The adapter config as JSON, or `@file`. `--url` and `--header` change the keys they set.
- `--auth`: The adapter credentials as JSON, or `@file`. For HTTP: `{"headers": {"Authorization": "Bearer ..."}}`. The API never returns them, so keep credentials here, not in `--header`.
- `--event-type`: An event type, or `<entity>.*` for every event of an entity. Repeat for more. Leave it out to receive every event.
- `--attribute`: An attribute value that the event's entity must have had, `key=value`. Repeat for more. Leave it out to receive events of any attributes.
- `--condition`: A jq expression run on each selected CloudEvent. The event is sent only when it outputs `true`.

Use `mass event-destination preview` to check event types, attributes, and a condition before you create a destination with them.

## Examples

```bash
# Send production deployment events to a URL
mass event-destination create \
  --name "Production deployments" \
  --url https://hooks.example.com/massdriver \
  --event-type 'deployment.*' \
  --attribute md-environment=prod

# Send credentials from a file
mass event-destination create \
  --name "Audit export" \
  --url https://hooks.example.com/massdriver \
  --auth @auth.json
```


```
mass event-destination create [flags]
```

### Options

```
      --adapter string           How the destination receives events (default "HTTP")
      --attribute stringArray    Attribute the event's entity must have had, key=value; repeat for more
      --auth string              Adapter credentials as JSON, or @file. Never returned by the API
      --condition string         jq expression that must output true for the event to be sent
      --config string            Adapter config as JSON, or @file
      --event-type stringArray   Event type, or <entity>.* for every event of an entity; repeat for more
      --header stringArray       Header sent with each event, Name=value (sets headers in the HTTP config, which the API returns; put credentials in --auth); repeat for more
  -h, --help                     help for create
      --name string              Name of the destination, unique in the organization
      --url string               URL to post events to (sets url in the HTTP config)
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass event-destination](/cli/commands/mass_event-destination)	 - Event destination management
