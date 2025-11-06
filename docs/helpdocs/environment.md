# Manage Environments

[Environments](https://docs.massdriver.cloud/concepts/environments) are the workspaces that bundles will be deployed to.

Environments can be modeled by application stage (production, staging, development), by region (prod-usw, prod-eu), and even ephemerally per developer (alice-dev, bob-dev).

## Commands

- `export`: Export an environment to local filesystem
- `get`: Retrieve environment details and configuration
- `list`: List all environments in a project
- `default`: Set an artifact as the default connection for an environment
