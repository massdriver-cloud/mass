# Print the Configuration File Path

Prints where the CLI reads its configuration from, so you can open or back up the file directly.

The location is `$XDG_CONFIG_HOME/massdriver/config.yaml` when `XDG_CONFIG_HOME` is set, and `~/.config/massdriver/config.yaml` otherwise.

## Usage

```bash
mass config path
```

## Examples

```bash
# Print the path
mass config path

# Open the file in your editor
$EDITOR "$(mass config path)"
```
