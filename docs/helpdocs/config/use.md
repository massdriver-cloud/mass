# Set the Active Profile

Makes a profile the one every `mass` command uses. The choice is written to the `current_profile` key in the config file, so it persists across shells and sessions.

`MASSDRIVER_PROFILE` and the `--profile` flag both take precedence over this setting.

## Usage

```bash
mass config use <profile>
```

## Examples

```bash
# Switch to the staging profile
mass config use staging

# Run a single command against a different profile
mass project list --profile production
```
