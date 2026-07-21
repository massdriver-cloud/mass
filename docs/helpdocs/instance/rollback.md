# Roll an Instance Back

Proposes a return to a past deployment's exact state. Given a historical deployment — which must be a COMPLETED PROVISION — this creates a new PROPOSED PROVISION deployment that snapshots the source deployment's params, connection wiring, bundle version, and release.

The returned proposal goes through the normal review flow: approve it with `mass deployment approve` or discard it with `mass deployment reject`. On approval, the instance is pinned to the source deployment's exact bundle version, params, and connection snapshot — overriding whatever release is currently configured.

## Usage

```bash
mass instance rollback <deployment-id>
```

Where `<deployment-id>` is the historical COMPLETED PROVISION deployment to return to.

## Examples

```bash
# Propose rolling back to a known-good past deployment
mass instance rollback 12345678-1234-1234-1234-123456789012

# Then approve it to apply
mass deployment approve <proposed-deployment-id>
```
