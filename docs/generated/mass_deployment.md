---
id: mass_deployment.md
slug: /cli/commands/mass_deployment
title: Mass Deployment
sidebar_label: Mass Deployment
---
## mass deployment

Manage deployments

### Synopsis

# Deployments

A deployment is a single infrastructure provisioning operation against an instance — either `PROVISION` (apply), `DECOMMISSION` (tear down), or `PLAN` (dry run). Deployments are immutable once created.

Use these commands to inspect deployment history and logs:

- `mass deployment list <instance-id>` — list recent deployments for an instance
- `mass deployment get <deployment-id>` — show details for a single deployment
- `mass deployment logs <deployment-id>` — print log output from a deployment
- `mass deployment abort <deployment-id>` — abort a pending, approved, or running deployment


### Options

```
  -h, --help   help for deployment
```

### SEE ALSO

* [mass](/cli/commands/mass)	 - Massdriver Cloud CLI
* [mass deployment abort](/cli/commands/mass_deployment_abort)	 - Abort a pending, approved, or running deployment
* [mass deployment get](/cli/commands/mass_deployment_get)	 - Get a deployment by ID
* [mass deployment list](/cli/commands/mass_deployment_list)	 - List deployments for an instance (most recent first)
* [mass deployment logs](/cli/commands/mass_deployment_logs)	 - Stream the log output from a deployment
