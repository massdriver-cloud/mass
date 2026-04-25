# Link Two Components

Creates a design-time link between two components: the source component's output (`fromField`) is wired to the destination component's input (`toField`). Both components must live in the same project. Massdriver validates that the resource types are compatible before creating the link.

At deploy time, each link becomes a **connection** that carries the actual resource data between instances.

## Usage

```shell
mass component link <from-component>.<from-field> <to-component>.<to-field> \
  [--from-version <version-constraint>] [--to-version <version-constraint>]
```

Versions default to `latest`.

## Examples

```shell
# Link the database's authentication output to the app's database input, with latest versions
mass component link ecomm-db.authentication ecomm-app.database

# Pin version constraints on both sides
mass component link ecomm-db.authentication ecomm-app.database \
  --from-version ~1.0 --to-version ~2.0
```
