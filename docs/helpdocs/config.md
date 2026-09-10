# Manage Configuration Profiles

A profile holds the organization, credential, and API URL used to talk to Massdriver. Profiles live in `~/.config/massdriver/config.yaml` (or `$XDG_CONFIG_HOME/massdriver/config.yaml`), and these commands read and write that file for you.

Use separate profiles to work across organizations or against a self-hosted installation without re-editing the file each time.

## Choosing a profile

A profile is selected in this order, highest precedence first:

1. The `--profile` flag
2. The `MASSDRIVER_PROFILE` environment variable
3. The `current_profile` key in the config file, set by `mass config use`
4. The profile named `default`

The first three name a profile explicitly, so a name that isn't in the file is an error rather than a silent fallback. The `default` fallback may be absent — that is how credentials supplied entirely through environment variables resolve.

## Getting started

```bash
mass config add default --org acme --api-key mds_xxx
mass config list
```
