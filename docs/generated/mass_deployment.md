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

Use these commands to inspect and manage deployments:

- `mass deployment list <instance-id>` — list recent deployments for an instance
- `mass deployment get <deployment-id>` — show details for a single deployment
- `mass deployment logs <deployment-id>` — print log output from a deployment
- `mass deployment abort <deployment-id>` — abort a pending, approved, or running deployment
- `mass deployment approve <deployment-id>` — approve a proposed deployment, releasing it to run
- `mass deployment reject <deployment-id>` — reject a proposed deployment, discarding it permanently

### Options

```
  -h, --help   help for deployment
```

### SEE ALSO

* [mass](/cli/commands/mass)	 - Massdriver Cloud CLI
* [mass deployment abort](/cli/commands/mass_deployment_abort)	 - Abort a pending, approved, or running deployment
* [mass deployment approve](/cli/commands/mass_deployment_approve)	 - Approve a proposed deployment, releasing it to run
* [mass deployment get](/cli/commands/mass_deployment_get)	 - Get a deployment by ID
* [mass deployment list](/cli/commands/mass_deployment_list)	 - List deployments for an instance (most recent first)
* [mass deployment logs](/cli/commands/mass_deployment_logs)	 - Stream the log output from a deployment
* [mass deployment reject](/cli/commands/mass_deployment_reject)	 - Reject a proposed deployment, discarding it permanently
