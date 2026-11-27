# Dashboard

A live view of your organization in the terminal.

**Constellation** places every environment you can view on a map, linked to the environments whose resources it uses and the environment it was forked from. Sparks travel each link from the producer to the consumer. Rings show the status of each environment's instances, a firing alarm pulses red, and a running deployment sweeps around its ring. Select an environment to see its instances, alarms, deployments, upgrades, the environments on either side of it, and its blast radius: everything downstream that its trouble could reach. With cost on, each ring is sized by last month's spend.

**Runtimes** shows which runtime versions your applications build on and your instances run, and what a patch, minor, or major release would leave behind.

**Alarms** lists every firing alarm with how far its blast radius reaches.

**Upgrades** lists every instance that can move to a newer bundle version.

A live ticker follows the audit log. When something happens, its environment lights up on the map, and a deployment event re-reads the organization.

**Replay** plays the audit log back, from five minutes to 24 hours ago, onto the constellation, so you can watch a rollout, an incident, or a busy afternoon move across your environments.

When the output is not a terminal, `mass dashboard` prints the same data as tables.

## Examples

```shell
mass dashboard
mass dashboard --refresh 10s --events 2s
mass dashboard --profile production | less
```
