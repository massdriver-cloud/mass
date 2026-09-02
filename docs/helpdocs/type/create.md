# Create Resource Type

Creates a new resource type OCI repository in your organization's catalog. The
repository starts empty; publish a version to it with
`mass resource-type publish`.

## Usage

```bash
mass resource-type create <name>
```

## Examples

```bash
# Create a resource type repository
mass resource-type create my-resource-type

# Create with custom attributes
mass resource-type create my-resource-type -a owner=data,service=database
```
