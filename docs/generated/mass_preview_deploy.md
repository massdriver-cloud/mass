---
id: mass_preview_deploy.md
slug: /cli/commands/mass_preview_deploy
title: Mass Preview Deploy
sidebar_label: Mass Preview Deploy
---
## mass preview deploy

Deploys a preview environment in your project

### Synopsis

# Deploy Preview Environment

Deploys a preview environment in your project.

Preview environments can be deployed arbitrarily from the command line or from pull requests and your CI/CD pipeline.

A configuration file with credential details and package parameters is required.

**Example:**

Deploy a project named "*ecomm*" specifying a CI context (`ci-context.json`) and a `preview.json` file from `mass preview init`.

```shell
mass preview init --output=./preview.json
mass preview deploy ecomm -c ./ci-context.json -p ./preview.json
```

## CI Context Push Events

GitHub and GitLab workflow events are officially support, but any CI Context file can be provided so long as it follows the format:

```json
{
  "pull_request": {
    "title": "Your title",
    "number": 1337
  }
}
```

`title` which will be used as the description of the environment and a "PR" `number` which is used in the environment's `name` and `slug`.

* [GitHub](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#push)
* [GitLab](https://docs.gitlab.com/ee/user/project/integrations/webhook_events.html#push-events)


```
mass preview deploy [flags]
```

### Options

```
  -c, --ci-context string   Path to GitHub Actions event.json (default "/home/runner/work/_temp/_github_workflow/event.json")
  -h, --help                help for deploy
  -p, --params string       Path to preview environment configuration file. This file supports bash interpolation. (default "./preview.json")
```

### SEE ALSO

* [mass preview](/cli/commands/mass_preview)	 - Create & deploy preview environments
