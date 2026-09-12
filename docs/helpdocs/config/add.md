# Add a Profile

Adds a profile to the configuration file. When run in a terminal, you are prompted for any required value you did not pass as a flag.

The credentials are checked against the Massdriver API before the profile is written, so a mistyped key or organization fails here rather than on your next command. Pass `--no-verify` to skip that check.

The first profile you add becomes the active profile.

## Usage

```bash
mass config add <profile>
```

## Examples

```bash
# Add a profile interactively
mass config add staging

# Add a profile non-interactively
mass config add staging --org acme --api-key mds_xxx

# Add a profile pointing at a self-hosted installation
mass config add onprem --org acme --api-key mds_xxx --url https://api.massdriver.example.com

# Add a profile and make it active
mass config add staging --org acme --api-key mds_xxx --use
```
