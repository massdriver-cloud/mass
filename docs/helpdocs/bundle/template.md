# Bundle Templates

Templates are boilerplate for extending the Massdriver platform with private infrastructure and applications. The boilerplates can be started with `mass bundle new` which will begin a questionnaire and interpolate your values into the boilerplate. From there you can customize the IaC in the src directory or UI in the massdriver.yaml file.

## Configuration

Templates are stored locally and configured via:

1. **Environment variable**: `MD_TEMPLATES_PATH`
2. **Config file**: `~/.config/massdriver/config.yaml`

### Config file example

```yaml
templates_path: /path/to/your/templates
```

## Available Commands

- `mass bundle template list` - List available templates in your configured templates directory

## Learn More

For more information on bundle templates, see the [Bundle Templates Guide](https://docs.massdriver.cloud/guides/bundle-templates).
