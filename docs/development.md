# Development

This document provides details to get you started with the development of the
Inventory system.

## Components

The Inventory system consists of the following components.

Please refer to the [Design Goals](./design.md) document for more details about
the overall design.

### API

The API service exposes collected and normalized data over a REST API.

### Persistence

For persisting the collected data the Inventory system uses a PostgreSQL
database.

The database models are based on [uptrace/bun](https://github.com/uptrace/bun).

### Worker

Workers are based on [hibiken/asynq](https://github.com/hibiken/asynq) and use
Redis (or any of the available alternatives such as Valkey and Redict) as a
message passing interface.

### Scheduler

The scheduler is based on [hibiken/asynq](https://github.com/hibiken/asynq) and
is used to trigger the execution of periodic tasks.

### Message Queue

Redis (or Valkey, or Redict) is used as a message queue for async communication
between the scheduler and workers.

### CLI

The CLI application is used for interfacing with the inventory system and
provides various sub-commands such as migrating the database schema, starting up
services, etc.

## Code Structure

The code is structured in the following way. Each data source (e.g. `aws`,
`gcp`, etc.) resides in it's own package. When introducing a new data source we
should follow the same pattern in order to keep the code consistent.

This is what the `pkg/aws` package looks like as of writing this document.

``` shell
pkg/aws
├── api        # API routes
├── client     # API clients
├── constants  # Constants
├── crud       # CRUD operations
├── models     # Data Models
├── tasks      # Async tasks
└── utils      # Utilities
```

## Tasks

Tasks are based on [hibiken/asynq](https://github.com/hibiken/asynq).

Each task should be defined in the respective `tasks` package of the data source
you are working on, e.g. `pkg/aws/tasks/tasks.go`.

An example task looks like this.

``` go
package tasks

import (
	"context"
	"fmt"

	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/hibiken/asynq"
)

// HandleSampleTask handles our sample task
func HandleSampleTask(ctx context.Context, t *asynq.Task) error {
	fmt.Println("handling sample task")

	return nil
}
```

The task must implement the
[asynq.Handler](https://pkg.go.dev/github.com/hibiken/asynq#Handler) interface.

Before we can use such a task in the workers, we need to register it. Only
tasks, which are registered will be considered by the workers, when we start
them up.

Task registration is done via the registry defined in
[pkg/core/registry](../pkg/core/registry).

The following example registers our sample task with the default task registry.

``` go
package tasks

import (
	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/hibiken/asynq"
)

// init registers our task handlers and periodic tasks with the registries.
func init() {
	// Task handlers
	registry.TaskRegistry.MustRegister("my-sample-task-name", asynq.HandlerFunc(HandleSampleTask))
}
```

You should also update the imports in
[cmd/inventory/tasks.go](../cmd/inventory/tasks.go) with an import statement
similar to the one below.

``` go
import _ "github.com/gardener/inventory/pkg/mydatasource/tasks"
```

We add this import solely for it's side-effects, so that task registration may
happen.

## Periodic Tasks

Periodic tasks are registered in a way similar to how we register worker tasks.

The following example registers a periodic task, which will be scheduled every
`30 seconds`.

In the `init()` function of your tasks package.

``` go
package tasks

import (
	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/hibiken/asynq"
)

// init registers our task handlers and periodic tasks with the registries.
func init() {
	// Periodic tasks
	sampleTask := asynq.NewTask("my-sample-task-name", nil)
	registry.ScheduledTaskRegistry.MustRegister("@every 10s", sampleTask)
}
```

Periodic tasks can be defined externally as well by adding an entry to the
scheduler configuration.

The following snippet adds a periodic job, which the scheduler will enqueue
every 1 hour.

``` yaml
# config.yaml
---
scheduler:
  jobs:
    - name: "my-sample-task"
      spec: "@every 1h"
      desc: "Foo does bar"
```

If a task requires a payload you can also specify the payload for it. For
example the following job will be invoked every 1 hour with the specified JSON
payload.

``` yaml
# config.yaml
---
scheduler:
  jobs:
    - name: "my-sample-task"
      spec: "@every 1h"
      desc: "Foo does bar"
      payload: >-
        {"foo": "bar"}
```

Make sure to check the [examples/config.yaml](../examples/config.yaml) file for
additional examples.

When running with multiple schedulers the example task above would be scheduled
by each instance of the scheduler, which would lead to tasks being duplicated
when being enqueued.

In order to avoid duplicating tasks by different instances of the scheduler you
should use the
[asynq.TaskID](https://pkg.go.dev/github.com/hibiken/asynq#TaskID) and
[asynq.Retention](https://pkg.go.dev/github.com/hibiken/asynq#Retention)
options.

For additional context and details, please refer to
[this asynq discussion](https://github.com/hibiken/asynq/discussions/376).

In the future we may explore different schedulers such as
[go-co-op/gocron](https://github.com/go-co-op/gocron), or use
[go-redsync/redsync](https://github.com/go-redsync/redsync) with `asynq`'s
scheduler.

## Local Environment

You can start a local environment using the provided
[Docker Compose](https://docs.docker.com/compose/) manifest.

The CLI tool uses a configuration file, which describes the settings of the
various components.

The following sample config file should get you started.

``` yaml
---
version: v1alpha1
debug: false

redis:
  endpoint: 127.0.0.1:6379

database:
  dsn: "postgresql://inventory:p4ssw0rd@localhost:5432/inventory?sslmode=disable"
  migration_dir: ./internal/pkg/migrations

worker:
  concurrency: 10
```

You can also use the [examples/config.yaml](../examples/config.yaml) file as a
starting point.

The config file can be set either by specifying `--config|--file` option when
invoking the `inventory` CLI, or via setting the `INVENTORY_CONFIG` env
variable.

In order to make development easier it is recommended to use
[direnv](https://direnv.net/) along with a `.envrc` file.

Here's a sample `.envrc` file, which you can customize.

``` shell
# .envrc
export INVENTORY_CONFIG=/path/to/inventory/config.yaml

# psql(1) env variables
export PGUSER=inventory
export PGDATABASE=inventory
export PGHOST=localhost
export PGPORT=5432
export PGPASSWORD=p4ssw0rd
```

The AWS tasks expect that you already have a shared configuration and
credentials files configured in `~/.aws/config` and `~/.aws/credentials`
respectively.

In order to start all services, execute the following command.

``` shell
docker compose up --build --remove-orphans
```

The services which will be started are summarized in the table below.

| Service     | Description                           |
|:------------|:--------------------------------------|
| `postgres`  | PostgreSQL database                   |
| `worker`    | Handles task messages                 |
| `scheduler` | Schedules tasks on regular basis      |
| `redis`     | Redis service used as a message queue |

Once the services are up and running you can access the following endpoints from
your local system.

| Endpoint       | Description       |
|:---------------|:------------------|
| localhost:5432 | PostgreSQL server |
| localhost:6379 | Redis server      |
