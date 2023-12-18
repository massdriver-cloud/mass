---
id: mass_server.md
slug: /cli/commands/mass_server
title: Mass Server
sidebar_label: Mass Server
---
## mass server

Start the bundle development server

### Synopsis

Start the bundle development server. If no port is supplied an ephemeral port will be used

```
mass server [flags]
```

### Options

```
  -d, --directory string   directory for the massdriver bundle, will default to the directory the server is ran from
  -h, --help               help for server
      --log-level string   Set the log level for the server. Options are [debug, info, warn, error] (default "info")
  -p, --port string        port for the server to listen on (default "8080")
```

### SEE ALSO

* [mass](/cli/commands/mass)	 - Massdriver Cloud CLI
