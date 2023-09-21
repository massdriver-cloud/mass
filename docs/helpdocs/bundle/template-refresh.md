# List Available Templates.

Sync local templates cache with the [official Massdriver templates repository](https://github.com/massdriver-cloud/application-templates).

The cache directory can be changed by setting the `MD_TEMPLATES_PATH` to your desired directory.

## Examples

```shell
mass bundle template refresh
```

### Override Local Template Cache Directory

By default templates are locally cached at `$HOME/.massdriver`.

```shell
export MD_TEMPLATES_PATH="$HOME/custom/"
mass bundle template refresh
```

### Custom Application Templates

You can also manage your own application templates for your teams.

They are expected to be hosted at a git accessible HTTP address. A simple naming convention and path structure is required. Templates must be top level directories in your repo with a `massdriver.yaml` file.

Like:

- `express-js/massdriver.yaml`
- `ruby-on-rails/massdriver.yaml`
- `company-name-standard-golang-service/massdriver.yaml`

Official templates can be used as a reference:

* https://github.com/massdriver-cloud/application-templates/tree/main/rails-kubernetes
* https://github.com/massdriver-cloud/application-templates/tree/main/aws-lambda

#### Overriding Template Source Repos

`MD_TEMPLATES_SRCS` can be set to a comma-separated list of GitHub repo URLs.

```shell
MD_TEMPLATES_SRCS="https://github.com/foo-corp/api-templates,https://github.com/bar-corp/ml-templates"
```

Full example:

```shell
export MD_TEMPLATES_PATH="$HOME/custom/"
# Note you'll need to explictly add Massdriver official templates in if youd like to continue using them when overridding
export MD_TEMPLATES_SRCS="https://github.com/my-org/templates,https://github.com/massdriver-cloud/application-templates"
mass bundle template refresh
```
