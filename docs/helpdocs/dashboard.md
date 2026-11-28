# Dashboard

A live view of your organization in the terminal.

The map centers on one environment: the one you name with `--env`, or else the one with the most recent deployment. It reads that environment and the ones it shares resources with, out to `--depth` hops. Environments past the edge show as 🔭 out of view. Select one and the view moves there: the focus glides to the middle and the environments around it come into frame. `/` finds an environment by name anywhere in the organization.

**Constellation** places those environments on a map, linked to the environments whose resources they use and the environment they were forked from. Sparks travel each link from the producer to the consumer. Rings show the status of each environment's instances, a firing alarm pulses red, and a running deployment sweeps around its ring. Select an environment to see its instances, alarms, deployments, upgrades, the environments on either side of it, and its blast radius: everything downstream that its trouble could reach. With cost on, each ring is sized by last month's spend. `f` follows the selected environment's deployment logs in the side panel: the deployment running there, or else the most recent one. Lines stream in as they are written, and the panel keeps the final status when the deployment ends. Streaming needs a personal access token; with a service account the logs are read every two seconds instead.

On the map, `n` and `N` pick an instance in the selected environment's side panel. `y` approves the picked instance's deployment that is awaiting approval (⏸), and `d` deploys it again with its current configuration. Both ask first, naming the instance, its environment, and how many environments and instances depend on it downstream; `y` or `enter` goes ahead, `c` shows the configuration of the deployment awaiting approval, and any other key cancels. `--read-only` turns both off.

**Runtimes** shows which runtime versions your applications build on and your instances run, and what a patch, minor, or major release would leave behind.

**Alarms** lists every firing alarm in view with how far its blast radius reaches.

**Upgrades** lists every instance in view that can move to a newer bundle version.

**Deployments** is where deployments get decided. It lists the deployments of the instances in view: first those awaiting approval, with who proposed each one, how long ago, and their message; then those in flight and how long they have run; then, dimmer, the 20 most recent to finish. `enter` shows the selected deployment's environment on the map, and `f` follows its logs across the whole screen. `c` shows its configuration: the message, the bundle version, and each param it changes since the instance's last completed provision, with anything that looks like a secret masked. `y` approves a deployment awaiting approval, and `c` in that confirmation shows its configuration before you decide. `x` rejects one; type a reason before pressing `enter` and it is kept in the deployment's stash under `review/rejection`. If the reason cannot be saved, the deployment is still rejected and the dashboard says so. `--read-only` turns off `y` and `x` too.

A live ticker follows the audit log. When something happens, its environment lights up on the map, and a deployment event re-reads what is in view.

**Replay** plays the audit log back, from five minutes to 24 hours ago, onto the map, so you can watch a rollout, an incident, or a busy afternoon move across your environments.

The dashboard follows the stash as a feed. It reads every stash in view once, then streams each key as it is set or deleted. Streaming needs a personal access token; with a service account, or while a stream reconnects, the dashboard reads the stash every few seconds instead (`--stash`). Each change goes on the ticker and lights up its environment. Widgets show where they belong: `md/v1/health` colors an instance, `md/v1/usage` draws its usage bar, `md/v1/traffic` sets the pace of the sparks on the links an instance sends on and labels them with requests per second, latency, and errors, `md/v1/banner` runs across the top of the map and flags its environment, and `md/v1/links` puts its dashboards and runbooks in the side panel. Your annotations show too: widgets you pinned to any stash key, and notes rendered from their Liquid templates. Annotations need a personal access token; a service account has none.

With `--watch`, an alarm that starts firing takes the view to its environment, locks on, shows its blast radius under a banner that counts up, and rings the terminal with a desktop notification. When it clears, the banner says how long it lasted and the view goes back where it was.

`w` writes up an incident: the one behind the selected alarm or environment, or else the most recent. It writes two files to the current directory, named for the alarmed instance, the alarm, and the minute the incident started, such as `incident-ecomm-prod-database-cpu-high-20261126-1402.md` and `incident-ecomm-prod-database-cpu-high-20261126-1402.json`. The markdown has how long the incident lasted, the alarm and its message, its blast radius, a timeline of the deployments, alarms, and stash changes in that blast radius from shortly before it fired until it cleared, and sections for the root cause and follow-ups. The JSON is a recording: the environments in view when it fired, their stash, and that timeline. `--playback` opens the dashboard on a recording, offline, and replays it on the map; `space` pauses and `[` and `]` change the speed.

`--theme` changes how the map is drawn: `constellation` (the default) scatters environments like stars, `metro` lays them out as stations on transit lines with producers to the left, `radar` puts the focus at the center of a sweeping scope with each environment on the ring as many hops out as it is, and `plain` lists environments by project, one line each, with their health, alarms, deploys, and upgrades, and nothing moving. `v` switches between them. `--palette` changes the colors and backdrop: `night` (the default), `rain`, and `blueprint`. `p` switches between them. `--render blocks` fills the map's dots in as solid blocks, so links and rings draw as solid lines rather than dotted ones. It suits a terminal that draws block characters itself, such as Ghostty; braille works in any terminal.

`e` saves the map as it stands to an SVG in the current directory, named for the organization and the minute, such as `constellation-acme-20261126-0905.svg`. It keeps the theme, the palette, and the selection, and draws in vectors, so it stays sharp on a slide or in an incident write-up. The constellation theme also draws the blast radius, alarm pulses, and traffic; metro and radar draw the blast radius in their links.

When the output is not a terminal, `mass dashboard` prints the same data as tables.

## Examples

```shell
mass dashboard
mass dashboard --env ecomm-prod --depth 2
mass dashboard --refresh 10s --events 2s --stash 1s
mass dashboard --watch
mass dashboard --read-only
mass dashboard --playback incident-ecomm-prod-database-cpu-high-20261126-1402.json
mass dashboard --theme radar --palette rain
mass dashboard --profile production | less
```
