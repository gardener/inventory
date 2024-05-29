# Database

The persistence layer used by the Gardener Inventory is
[PostgreSQL](https://www.postgresql.org/).

# Data Model

The high-level overview of the data model followed by the Gardener Inventory
system looks like this.

![Data Model](../images/data-model.png)

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

Example model would look like this.

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

# Migrations

Database migrations are managed by the CLI tool.

## Initialize Database

Before we apply any migrations we need to initialize the database tables.

The following command expects that you already have a configured
[connection string](https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING)
to the database we will be migrating via the `DSN` environment variable.

``` shell
inventory db init
```

If you want to explicitely specify an alternate connection string you can do
that by using the `--dsn` option, e.g.

``` shell
inventory db --dsn postgres://user:p4ss@localhost:5432/foo init
```

## Migrations Status

Check the database migration status by executing the following command.

``` shell
inventory db status
```

Sample output looks like this.

``` text
pending migration(s): 0
database version: group #1 (20240522121536_aws_add_region)
database is up-to-date
```

## View Pending Migrations

Apply any pending migrations by executing the following command.

``` shell
inventory db pending
```

## View Applied Migrations

In order to view the list of applied migrations you need to execute the
following command.

``` text
inventory db applied
```

## Create New Migrations

In order to create a new migration sequence execute the following command, which
will generate an `up` and `down` migration file for you.

``` shell
inventory db create <description-of-my-change>
```

## Apply Migrations

In order to apply all pending migrations you should execute the following
command.

``` shell
inventory db migrate
```

## Rolling Back Migrations

Rolling back migrations is done via the `inventory db rollback` command.

``` shell
inventory db rollback
```

The command above will rollback the last applied migration group.
