# Mass CLI

[![GitHub license](https://img.shields.io/github/license/massdriver-cloud/mass)](https://github.com/massdriver-cloud/mass/blob/main/LICENSE)
[![GitHub issues](https://img.shields.io/github/issues/massdriver-cloud/mass)](https://github.com/massdriver-cloud/mass/issues)
[![GitHub release](https://img.shields.io/github/release/massdriver-cloud/mass.svg)](https://GitHub.com/massdriver-cloud/mass/releases/)
[![Go Report Card](https://goreportcard.com/badge/github.com/massdriver-cloud/mass)](https://goreportcard.com/report/github.com/massdriver-cloud/mass)
[![Go Reference](https://pkg.go.dev/badge/github.com/massdriver-cloud/mass.svg)](https://pkg.go.dev/github.commassdriver-cloud/mass)

The Mass CLI is a command line tool to manage applications and infrastructure on [Massdriver Cloud](https://massdriver.cloud).

Official [GitHub actions](https://github.com/massdriver-cloud/actions) are also available.

## Installation

### Pre-built Binaries

Pre-built binaries for the Mass CLI are available in the [Releases](https://github.com/massdriver-cloud/mass/releases) section of this repository.

<!--
### Homebrew

```sh
brew install mass
```
-->

### Go

```sh
go install github.com/massdriver-cloud/mass
```

## Usage

The `mass` command line tool provides a number of subcommands to interact with Massdriver Cloud. For detailed usage and examples, please see the [official documentation](https://docs.massdriver.cloud/cli/overview).

### Preview Environments

#### Initialize a Preview Environment Config File

The preview environment config file should be checked into your source repository. The `preview.json` file supports bash interpolation in the event you need to dynamically set values from your CI.

**Examples:**

`mass preview init $yourProjectSlug`

`mass preview init ecomm`

`mass preview init ecomm --output path/to/my/preview.json`

##### Preview Environment Config Files

The `preview.json` file serves two purposes in your preview environment:

1. describes which clouds and the authentication to use
2. sets the input parameters for _each_ of your packages

```js
{
  "credentials": {
    // Using an AWS IAM Role
    "massdriver/aws-iam-role": "00000000-0000-0000-0000-000000000000"
  },
  "packageParams": {
    "database": {
      "cpus": "1",
      "memory": "over9000GB"
    },
    "my-api": {
      "image": "evilcorp/api:$IMAGE_TAG"
    }
  }
}
```

#### Deploy Preview Environment

Deploys a preview environment in your project.

Preview environments can be deployed arbitrarily from the command line or from pull requests and your CI/CD pipeline.

A configuration file with credential details and package parameters is required.

**Example:**

Deploy a project named "*ecomm*" specifying a CI context (`ci-context.json`) and a `preview.json` file from `mass preview init`.

```shell
mass preview init --output=./preview.json
mass preview deploy ecomm -c ./ci-context.json -p ./preview.json
```

##### CI Context Push Events

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


### Deploy applications on Massdriver.

This application must be published as a [bundle](https://docs.massdriver.cloud/applications) to Massdriver first and be configured for a given environment (target).

#### Examples

<!--
![Finding an application slug in Massdriver Cloud](./application-slug.png)
-->

You can deploy an application using the _fully qualified name_ of the application or its `slug`.

The `slug` can be found by hovering over the application name in the Massdriver diagram.

**Using the fully qualified name**:

```shell
mass app deploy ecomm-prod-api
```

**Using the slug**:

```shell
mass app deploy ecomm-prod-api-x12g
```

For more info see [deploying](https://docs.massdriver.cloud/applications/deploying-application).

## Contributing

If you'd like to contribute to the Mass CLI, please refer to the [Contribution Guidelines](https://github.com/massdriver-cloud/mass/blob/main/CONTRIBUTING.md).

## License

The Mass CLI is open source software licensed under the [MIT license](https://github.com/massdriver-cloud/mass/blob/main/LICENSE).
