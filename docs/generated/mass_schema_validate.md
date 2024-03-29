---
id: mass_schema_validate.md
slug: /cli/commands/mass_schema_validate
title: Mass Schema Validate
sidebar_label: Mass Schema Validate
---
## mass schema validate

Validates a JSON document against a JSON Schema

### Synopsis

# Validation JSON Documents Against Schemas

This command is useful during development and CI to validate JSON documents & schemas generated by the Massdriver CLI.

Given the following `data.json` and `schema.json`:

```shell
mass schema validate --document=data.json --schema=schema.json
```

**data.json**

```json
{
  "firstName": "John",
  "lastName": "Doe",
  "age": 23
}
```

**schema.json**

```json
{
  "$id": "https://example.com/person.schema.json",
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Person",
  "type": "object",
  "properties": {
    "firstName": {
      "type": "string",
      "description": "The person's first name."
    },
    "lastName": {
      "type": "string",
      "description": "The person's last name."
    },
    "age": {
      "description": "Age in years which must be equal to or greater than zero.",
      "type": "integer",
      "minimum": 0
    }
  }
}
```


```
mass schema validate [flags]
```

### Options

```
  -d, --document string   Path to JSON document (default "document.json")
  -h, --help              help for validate
  -s, --schema string     Path to JSON Schema (default "./schema.json")
```

### SEE ALSO

* [mass schema](/cli/commands/mass_schema)	 - Manage JSON Schemas
