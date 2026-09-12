# Remove a Profile

Removes a profile from the configuration file. You are asked to confirm by typing the profile name unless `--force` is passed.

If the removed profile was the active one, `current_profile` is cleared so later commands fall back to the `default` profile instead of failing on a name that no longer exists.

## Usage

```bash
mass config remove <profile>
```

## Examples

```bash
# Remove a profile
mass config remove staging

# Remove without the confirmation prompt
mass config remove staging --force
```
