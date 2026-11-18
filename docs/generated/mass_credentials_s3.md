---
id: mass_credentials_s3.md
slug: /cli/commands/mass_credentials_s3
title: Mass Credentials S3
sidebar_label: Mass Credentials S3
---
## mass credentials s3

Print AWS credentials for the Massdriver S3 API

### Synopsis

# S3 Credentials

Prints AWS credentials for the Massdriver S3 API. Tools such as the AWS CLI, AWS SDKs, and rclone use them to read instance params and to propose deployments.

The profile's API key must be an access token (`md_...`).

- Access key ID: the first 12 characters of the access token
- Secret access key: the hex SHA-256 of the access token

## Usage

```bash
mass credentials s3 [--output process|env]
```

`--output process` (the default) prints JSON for the AWS `credential_process` setting. `--output env` prints shell exports, including `AWS_ENDPOINT_URL_S3`.

## Examples

```bash
# Set the credentials in the current shell
eval "$(mass credentials s3 --output env)"
aws s3 ls s3://acme/

# Use the credentials from an AWS profile
cat >> ~/.aws/config <<'EOF'
[profile massdriver]
credential_process = mass credentials s3
endpoint_url = https://api.massdriver.cloud/s3
EOF
aws s3 cp s3://acme/ecomm/prod/db.json - --profile massdriver
```


```
mass credentials s3 [flags]
```

### Examples

```
mass credentials s3 --output env
```

### Options

```
  -h, --help            help for s3
  -o, --output string   Output format (process, env) (default "process")
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass credentials](/cli/commands/mass_credentials)	 - Print credentials for other tools to reach Massdriver
