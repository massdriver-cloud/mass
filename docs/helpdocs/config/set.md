# Update a Profile

Updates settings on an existing profile. Only the values you pass are changed; everything else is left alone. Pass an empty string to remove a setting.

Credentials are verified against the Massdriver API before the change is written. Pass `--no-verify` to skip that check.

## Usage

```bash
mass config set <profile>
```

## Examples

```bash
# Rotate the API key
mass config set staging --api-key mds_yyy

# Point a profile at a self-hosted installation
mass config set staging --url https://api.massdriver.example.com

# Set where `mass bundle new` looks for templates
mass config set staging --templates-path ~/massdriver/templates

# Clear the templates path
mass config set staging --templates-path ""
```
