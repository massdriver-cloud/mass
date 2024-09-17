# Import IaC variables into Massdriver params

This command will scan the directories defined in the bundle steps, and attempt to identify variables declared in IaC, but not yet exposed as Massdriver params.

## Examples with Flags

By default, this command will prompt you for confirming import on every missing parameter:

```shell
mass bundle import
```

If you want to import all missing params without prompting (for bulk import or in automation), use the -a flag

```shell
mass bundle import -a
```
