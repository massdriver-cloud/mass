---
id: mass_dashboard.md
slug: /cli/commands/mass_dashboard
title: Mass Dashboard
sidebar_label: Mass Dashboard
---
## mass dashboard

Watch your environments, alarms, and runtimes live

### Synopsis

# Dashboard

A live view of your organization in the terminal.

**Constellation** places every environment you can view on a map, linked to the environments whose resources it uses and the environment it was forked from. Sparks travel each link from the producer to the consumer. Rings show the status of each environment's instances, a firing alarm pulses red, and a running deployment sweeps around its ring. Select an environment to see its instances, alarms, deployments, upgrades, the environments on either side of it, and its blast radius: everything downstream that its trouble could reach. With cost on, each ring is sized by last month's spend.

**Runtimes** shows which runtime versions your applications build on and your instances run, and what a patch, minor, or major release would leave behind.

**Alarms** lists every firing alarm with how far its blast radius reaches.

**Upgrades** lists every instance that can move to a newer bundle version.

A live ticker follows the audit log. When something happens, its environment lights up on the map, and a deployment event re-reads the organization.

**Replay** plays the audit log back, from five minutes to 24 hours ago, onto the constellation, so you can watch a rollout, an incident, or a busy afternoon move across your environments.

The dashboard follows the stash as a feed, reading it every few seconds (`--stash`). Each change goes on the ticker and lights up its environment. Widgets show where they belong: `md/health` colors an instance, `md/usage` draws its usage bar, `md/traffic` sets the pace of the sparks on the links an instance sends on and labels them with requests per second, latency, and errors, `md/banner` flags an environment, and `md/links` puts its dashboards and runbooks in the side panel. Your annotations show too: widgets you pinned to any stash key, and notes rendered from their Liquid templates. Annotations need a personal access token; a service account has none.

With `--watch`, an alarm that starts firing takes the view to its environment, locks on, shows its blast radius under a banner that counts up, and rings the terminal with a desktop notification. When it clears, the banner says how long it lasted and the view goes back where it was.

`--theme` changes the look: `constellation` (the default) draws on a starfield, `rain` on falling green streams, and `blueprint` on a drafting grid.

When the output is not a terminal, `mass dashboard` prints the same data as tables.

## Examples

```shell
mass dashboard
mass dashboard --refresh 10s --events 2s --stash 1s
mass dashboard --watch
mass dashboard --theme rain
mass dashboard --profile production | less
```


```
mass dashboard [flags]
```

### Options

```
      --events duration    How often to read new activity from the audit log; 0 turns the live ticker off (default 5s)
  -h, --help               help for dashboard
      --refresh duration   How often to re-read the organization; 0 reads once (default 30s)
      --stash duration     How often to read the stash for widgets, notes, and the live feed; each read covers every scope, so a short interval costs more; 0 reads it with each refresh (default 3s)
      --theme string       Look of the dashboard: constellation, blueprint, rain (default "constellation")
      --watch              Jump to any alarm that starts firing while the dashboard is open, and send a desktop notification
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass](/cli/commands/mass)	 - Massdriver Cloud CLI
