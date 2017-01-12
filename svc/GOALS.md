# svc
## Goals

- Standardize service deployments to have _one_ syntax and _one_ function for the following:
  1. Deployment
  2. Checking the status of a deployed service
  3. Killing off an old instance of the service

- Create a command line tool that deploys a service to a given provider
  given configuration in a simple yaml manifest (see example [here](https://github.com/Xe/tools/tree/master/svc/sample))

- Persist a mapping of service names -> identifier for keeping track of past deployments

## Subcommands

| cmd | what it does |
|:--- |:------------ |
| `spawn` | Launches a new instance of the given service name on the given backend |
| `ps` | Inquires the status of all known deployed services and displays them in a clever little grid |
| `create` | Creates a directory hierarchy at $SVCROOT for a new service by name |
| `remove` | Stops a service and undeploys it from a given backend |
| `cycle` | Pulls the latest image and restarts the service with the new image |
| `inspect` | Inspects a single service, outputting its state in json |

### `spawn`

Launches a new instance of the given service name on the given backend

Usage: `svc spawn [options] <servicename> <backend>`

Options:

| option | type | effect |
|:------ |:---- |:------ |
| `-kahled` | bool | Creates another instance of this service if one exists on any backend, fails if service is exclusive and already spawned |

### `ps`

Inquires the status of all known deployed services and displays them in a clever little grid

Usage `svc ps [options] [servicename]`

Options:

| option | type | effect |
|:------ |:---- |:------ |
| `-backend` | string | If set, only show results for services running on the given backend |
| `-match` | string | If set, regex-match on service details |
| `-format` | string | Pretty-print container status using a Go template |

### `create`

Creates a directory hierarchy at $SVCROOT for a new service by name

Usage: `svc create <servicename>`

### `remove`

Stops a service and removes it from a given backend

Usage: `svc remove <servicename>`

### `cycle`

Pulls the latest image and restarts the service with the new image

This command ***NEVER*** stops the old container until the new container is running and passes
healthchecks.

Usage: `svc cycle <servicename>`

### `inspect`

Inspects a single service from a single backend, outputting its state in json

By default this will output a list of the inspect state of all matching instances of a service
running on a particular backend.

Usage: `svc inspect <servicename> <backend>`
