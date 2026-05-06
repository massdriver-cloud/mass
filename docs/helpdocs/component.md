# Components

A project contains a **blueprint** — the design-time architecture of your infrastructure. Components are the slots in that blueprint, each backed by a bundle. **Links** describe how one component's output wires into another component's input.

At deploy time, each environment materializes the blueprint into live **instances** and **connections**.

Use these commands to manage components and links in a project's blueprint:

- `mass component add` — add a component to a project
- `mass component remove` — remove a component
- `mass component link` — link two components together
- `mass component unlink` — remove a link
