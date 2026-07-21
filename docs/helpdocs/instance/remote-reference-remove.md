# Remove an Instance Remote Reference

Removes a per-instance connection slot override, reverting the slot to the blueprint Link's wiring (or the environment default).

Like other configuration changes, the instance must not be in PROVISIONED or FAILED status.

## Usage

```bash
mass instance remote-reference remove <instance-id> <field>
```

Where `<field>` is the connection slot key whose override should be removed.

## Examples

```bash
mass instance remote-reference remove ecomm-prod-api database
```
