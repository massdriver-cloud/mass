# Remove a Component from a Project

Removes a component from a project's blueprint, along with all its links. Any deployed instances of this component must be decommissioned first.

## Usage

```shell
mass component remove <component-id>
```

The component ID is in the `<project-id>-<component-id>` format.

## Examples

```shell
mass component remove ecomm-db
```
