## mass application deploy

Deploy applications

### Synopsis

# Deploy applications on Massdriver.

This application must be published as a [bundle](https://docs.massdriver.cloud/applications) to Massdriver first and be configured for a given environment (target).

## Examples

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


```
mass application deploy <project>-<target>-<manifest> [flags]
```

### Options

```
  -h, --help   help for deploy
```

### SEE ALSO

* [mass application](mass_application.md)	 - Manage applications

###### Auto generated by spf13/cobra on 19-Sep-2023