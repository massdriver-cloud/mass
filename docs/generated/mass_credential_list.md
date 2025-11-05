---
id: mass_credential_list.md
slug: /cli/commands/mass_credential_list
title: Mass Credential List
sidebar_label: Mass Credential List
---
## mass credential list

List credentials

### Synopsis

# List Credentials

Lists all credential artifacts in your organization.

## Usage

```bash
mass credential list
```

## Examples

```bash
# List all credentials
mass credential list
```

The output displays a table with:
- **ID**: Unique identifier for the credential
- **Name**: Name of the credential artifact
- **Updated At**: Last update timestamp


```
mass credential list [flags]
```

### Options

```
  -h, --help   help for list
```

### SEE ALSO

* [mass credential](/cli/commands/mass_credential)	 - Credential management
