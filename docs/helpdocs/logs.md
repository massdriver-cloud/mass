# Get Deployment Logs

Retrieves and outputs the log stream for a specific deployment. The logs are dumped to stdout in their original format.

## Usage

```bash
mass logs <deployment-id>
```

Where `<deployment-id>` is the UUID of the deployment.

## Examples

```bash
# Get logs for a deployment
mass logs 12345678-1234-1234-1234-123456789012

# Pipe logs to a file
mass logs 12345678-1234-1234-1234-123456789012 > deployment.log
```

## Notes

- Logs are output to stdout in their original format
- This command does not support tailing/following logs - it dumps all available logs
- The deployment ID can be found in the Massdriver UI or from deployment-related commands
