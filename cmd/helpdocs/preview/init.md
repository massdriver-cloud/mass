# Initialize a Preview Environment Config File

The preview environment config file should be checked into your source repository. The `preview.json` file supports bash interpolation in the event you need to dynamically set values from your CI.

**Examples:**

`mass preview init $yourProjectSlug`

`mass preview init ecomm`

`mass preview init ecomm --output path/to/my/preview.json`

## Preview Environment Config Files

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
