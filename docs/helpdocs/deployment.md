# Deployments

A deployment is a single infrastructure provisioning operation against an instance — either `PROVISION` (apply), `DECOMMISSION` (tear down), or `PLAN` (dry run). Deployments are immutable once created.

Use these commands to inspect deployment history and logs:

- `mass deployment list <instance-id>` — list recent deployments for an instance
- `mass deployment get <deployment-id>` — show details for a single deployment
- `mass deployment logs <deployment-id>` — print log output from a deployment
