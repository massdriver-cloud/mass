# List Available Templates.

Sync local templates cache with the [official Massdriver templates repository](https://github.com/massdriver-cloud/application-templates). Custom directories can be set for development by
setting the `MD_TEMPLATES_PATH` to your desired directory

## Examples

```shell
mass bundle template refresh
```

### With developer override

```shell
export MD_TEMPLATES_PATH=$HOME/custom/
mass bundle template refresh
```
