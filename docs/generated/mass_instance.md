---
id: mass_instance.md
slug: /cli/commands/mass_instance
title: Mass Instance
sidebar_label: Mass Instance
---
## mass instance

Manage instances of IaC deployed in environments.

### Synopsis

# Instances

[Instances](https://docs.massdriver.cloud/concepts/instances) are deployment of infrastructure-as-code modules on your environment canvas.

Instances are used to:
- Deploy infrastructure components
- Configure application services
- Manage environment-specific settings
- Connect different components together


### Options

```
  -h, --help   help for instance
```

### SEE ALSO

* [mass](/cli/commands/mass)	 - Massdriver Cloud CLI
* [mass instance copy](/cli/commands/mass_instance_copy)	 - Copy an instance's configuration to another instance of the same component
* [mass instance deploy](/cli/commands/mass_instance_deploy)	 - Deploy instances
* [mass instance destroy](/cli/commands/mass_instance_destroy)	 - Destroy (decommission) an instance
* [mass instance export](/cli/commands/mass_instance_export)	 - Export instances
* [mass instance get](/cli/commands/mass_instance_get)	 - Get an instance
* [mass instance list](/cli/commands/mass_instance_list)	 - List instances in an environment
* [mass instance orphan](/cli/commands/mass_instance_orphan)	 - Orphan an instance (reset to INITIALIZED, optionally clearing state locks)
* [mass instance version](/cli/commands/mass_instance_version)	 - Set instance version
