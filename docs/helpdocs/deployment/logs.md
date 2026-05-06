# Get Deployment Logs

Prints the log output emitted by a deployment, oldest first. Each batch is a single worker flush; a batch's message may contain multiple newline-separated lines.

## Usage

```shell
mass deployment logs <deployment-id>
```

## Examples

```shell
mass deployment logs 12345678-1234-1234-1234-123456789012
```
