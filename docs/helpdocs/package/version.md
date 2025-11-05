# Set package version

Set the version or release channel for a package in Massdriver.

## Examples

Set the version for a package using the `slug@version` format:

```shell
mass package version api-prod-db@latest
```

Set the version with a specific release channel:

```shell
mass package version api-prod-db@latest --release-channel development
```

The `slug` can be found in the package info panel. The package slug is a combination of the `<project-slug>-<env-slug>-<manifest-slug>`.

## Release Channels

- `stable` (default): Package receives only stable releases
- `development`: Package receives both stable and development releases

## Version Format

The version can be:
- A semantic version (e.g., `1.2.3`)
- A version constraint (e.g., `~1.2`, `~1`)
- A release channel (e.g., `latest`)
