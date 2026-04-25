# Unlink Two Components

Removes a design-time link between two components, identified by the `fromComponent.fromField → toComponent.toField` wiring. Existing connections in deployed environments are not affected until the next deployment.

## Usage

```shell
mass component unlink <from-component>.<from-field> <to-component>.<to-field>
```

## Examples

```shell
mass component unlink ecomm-db.authentication ecomm-app.database
```
