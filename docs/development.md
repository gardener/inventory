# Development

This document provides details to get you started with the development of the
Inventory system.

The development in summary is:

1. Define models and then register them
2. Create schema migration for the models
3. Define tasks and register them
4. Configure retention policy for the models
5. Define scheduler entries for any periodic tasks
6. Add test cases

For more details on each point, please read the rest of this document.

# Components

The Inventory system consists of the following components.

Please refer to the [Design Goals](./design.md) document for more details about
the overall design.

## API

The API service exposes collected and normalized data over a REST API.

## Persistence

For persisting the collected data the Inventory system uses a PostgreSQL
database.

The database models are based on [uptrace/bun](https://github.com/uptrace/bun).

## Worker

Workers are based on [hibiken/asynq](https://github.com/hibiken/asynq) and use
Redis (or any of the available alternatives such as Valkey and Redict) as a
message passing interface.

## Scheduler

The scheduler is based on [hibiken/asynq](https://github.com/hibiken/asynq) and
is used to trigger the execution of periodic tasks.

## Message Queue

Redis (or Valkey, or Redict) is used as a message queue for async communication
between the scheduler and workers.

## CLI

The CLI application is used for interfacing with the inventory system and
provides various sub-commands such as migrating the database schema, starting up
services, etc.

# Code Structure

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

# Models

The database models are based on [uptrace/bun](https://github.com/uptrace/bun).

The following sections provide additional details about naming conventions and
other hints to follow when creating a new model, or updating an existing one.

## Base Model

The [pkg/core/models](../pkg/core/models) package provides base models, which are
meant to be used by other models.

Make sure that you embed the [pkg/core/models.Model](./pkg/core/models) model
into your models, so that we have a consistent models structure.

In additional to our core model, we should also embed the
[bun.BaseModel](https://pkg.go.dev/github.com/uptrace/bun#BaseModel) model,
which would allow us to customize the model further, e.g. specifying a
different table name, alias, view name, etc.

Customizing the table name, alias and view name for a model can only be
configured on the `bun.BaseModel`. See the
[Struct Tags](https://bun.uptrace.dev/guide/models.html#struct-tags) section from the
[uptrace/bun](https://bun.uptrace.dev/guide/) documentation.

## Example Model

An example model would look like this.

``` go
package my_package

import (
	coremodels "github.com/gardener/inventory/pkg/core/models"
	"github.com/uptrace/bun"
)

// MyModel does something
type MyModel struct {
	bun.BaseModel `bun:"table:my_table_name"`
	coremodels.Model

	Name string `bun:"name,notnull,unique"`
}
```

Make sure to check the documentation about [defining
models](https://bun.uptrace.dev/guide/models.html) for additional information
and examples.

Also, once you've created the model you should create a migration for it.

``` shell
inventory db create <description-of-your-migration>
```

The command above will create two migration files. Edit the files and describe
the schema of your model, then commit them to the repo.

Finally, you should register your model with the default models registry.

``` go
func init() {
	// Register the models with the default registry
	registry.ModelRegistry.MustRegister("foo:model:bar", &MyModel{})
}
```

The naming convention we follow when defining new models is
`<datasource>:model:<modelname>`. For example, if you are defining a new AWS
model called `Foo` you should register the model using the `aws:model:foo` name.

## Model Retention

Each data model registers itself with the
[default model registry](../pkg/core/registry).

In order to keep the database clean from stale records the Inventory system runs
a periodic housekeeper task, which cleans up records based on a retention
period.

In order to define a retention period for an object you should update the
`common:task:housekeeper` task payload in your
[config.yaml](../examples/config.yaml) and add an entry for your object.

The following example snippet configures retention for the `foo:model:bar`
model, which will remove records that were not updated in the last 4 hours.

In this configuration the housekeeper task will be invoked every 1 hour.

``` yaml
scheduler:
  jobs:
    # The housekeeper takes care of cleaning up stale records
    - name: "common:task:housekeeper"
      spec: "@every 1h"
      payload: >-
        retention:
          - name: "foo:model:bar"
            duration: 4h
```

## Link / Nexus Tables

Relationships between models in the database are established with the help of
[link tables](https://en.wikipedia.org/wiki/Associative_entity).

Even though some relationships might be one-to-one, or one-to-many we still use
separate link tables to connect the models for a few reasons.

- Keep the codebase and database more consistent and cleaner
- Be able to enhance the model with additional attributes, by having properties
  on the link table

Since data collection runs concurrently and indendently from each other, linking
usually happens on a later stage, e.g. when all relevant data is collected.

Another benefit of having a separate link table is that we can extend the model
by having additional properties on the link table itself, and this way we don't
have to modify the base models, e.g. have columns which identify when the link
was created, updated, etc. This allows us to keep our collectors simple at the
same time.

There are two kinds of _link tables_ we use in the Inventory system, and both
serve the same purpose, but are called differently.

When we create links between models within the same data source (e.g. AWS) we create
relationships between the models by following this naming convention:

- `l_<datasource>_<model_a>_to_<model_b>`

For instance, the following link table contains the relationships between the
AWS Region and the AWS VPCs, where both models are defined in the same Go
package.

- `l_aws_region_to_vpc`

When creating cross-package relationships we create a
[nexus](https://en.wiktionary.org/wiki/nexus) table instead, which follows this
naming convention.

- `n_<datasource_a>_<model_a>_to_<datasource_b>_<model_b>`

For instance, the following _nexus_ table contains the relationships between the
AWS VPCs and Gardener Shoots.

- `n_aws_vpc_to_g_shoot`

Both tables -- _link_ and _nexus_ -- serve the same purpose, but are named
differently in order to make it easier to distinguish between _in-package_ and
_cross-package_ relationships.

# Tasks

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
[cmd/inventory/init.go](../cmd/inventory/init.go) with an import statement
similar to the one below.

``` go
import _ "github.com/gardener/inventory/pkg/mydatasource/tasks"
```

We add this import solely for it's side-effects, so that task registration may
happen.

# Periodic Tasks

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

The naming convention we use when defining new tasks is
`<datasource>:task:<taskname>`. For example, if you are creating a new
AWS-specific task called `foo`, you should register the task with the following
name: `aws:task:foo`.

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

# Local Environment

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
```

The AWS tasks expect that you already have a shared configuration and
credentials files configured in `~/.aws/config` and `~/.aws/credentials`
respectively.

In order to start all services, execute the following command.

``` shell
docker compose up --build --remove-orphans
```

The services which will be started are summarized in the table below.

| Service      | Description                               |
|:-------------|:------------------------------------------|
| `postgres`   | PostgreSQL database                       |
| `worker`     | Handles task messages                     |
| `scheduler`  | Schedules tasks on regular basis          |
| `redis`      | Redis service used as a message queue     |
| `dashboard`  | Asynq UI dashboard and Prometheus metrics |
| `grafana`    | Grafana instance                          |
| `prometheus` | Prometheus instance                       |

Once the services are up and running you can access the following endpoints from
your local system.

| Endpoint                      | Description                 |
|:------------------------------|:----------------------------|
| localhost:5432                | PostgreSQL server           |
| localhost:6379                | Redis server                |
| http://localhost:8080/        | Dashboard UI                |
| http://localhost:8080/metrics | Prometheus Metrics endpoint |
| http://localhost:3000/        | Grafana UI                  |
| http://localhost:9090/        | Prometheus UI               |

# Testing

In order to run the unit tests execute the following command.

``` shell
make test
```
