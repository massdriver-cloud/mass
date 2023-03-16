# Push Container Images To Cloud Repositories

Create registries, repositories and push images via the Massdriver CLI. Massdriver will build a Docker registry if it does not exist in the region in which you are pushing an image, create a repository in that region's registry and finally push a tagged version of the image to that repository.

## Examples

```bash
mass image push massdriver-cloud/massdriver \
    --region us-east-1 \
    --artifact xxxx \
    --tag v1
```

In the above example massdriver would create a registry with the namespace provided, and push your built container as the image name in that registry. The artifact ID is a unique idenifier for a credential artifact in Massdriver that is authorized to access the cloud account you are pushing the image to. The tag is the image tag which can be used to signal container orchestration systems which version of the image to pull.
