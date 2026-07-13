# Deployments

A deployment is a single infrastructure provisioning operation against an instance — either `PROVISION` (apply), `DECOMMISSION` (tear down), or `PLAN` (dry run). Deployments are immutable once created.

Use these commands to inspect and manage deployments:

- `mass deployment list <instance-id>` — list recent deployments for an instance
- `mass deployment get <deployment-id>` — show details for a single deployment
- `mass deployment logs <deployment-id>` — print log output from a deployment
- `mass deployment abort <deployment-id>` — abort a pending, approved, or running deployment
- `mass deployment approve <deployment-id>` — approve a proposed deployment, releasing it to run
- `mass deployment reject <deployment-id>` — reject a proposed deployment, discarding it permanently
