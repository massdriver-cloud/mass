---
id: mass_credentials.md
slug: /cli/commands/mass_credentials
title: Mass Credentials
sidebar_label: Mass Credentials
---
## mass credentials

Print credentials for other tools to reach Massdriver

### Synopsis

# Credentials for Other Tools

Prints credentials that other tools use to reach Massdriver. Each credential comes from the access token in the active profile.

## Commands

- `mass credentials s3`: AWS credentials for the Massdriver S3 API


### Options

```
  -h, --help   help for credentials
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass](/cli/commands/mass)	 - Massdriver Cloud CLI
* [mass credentials s3](/cli/commands/mass_credentials_s3)	 - Print AWS credentials for the Massdriver S3 API
