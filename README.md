# inventory

Gardener Inventory repo.

TODO: Additional details and diagrams about the project

# Requirements

- Go 1.22.x or later
- [Redis](https://redis.io/)
- [PostgreSQL](https://www.postgresql.org/)
- [Atlas](https://atlasgo.io/) for schema migrations

Instead of using Redis because of their recent licence change, please consider
using their drop-in replacements such as
[Valkey](https://github.com/valkey-io/valkey) or [Redict](https://redict.io).

# Design Goals

TODO: Document me

# Components

The Gardener Inventory consists of the following components.

## API

TODO: Document me

## Persistence

For persisting the collected data the Gardener Inventory will use a PostgreSQL
database.

The database models are built on top of [GORM](https://gorm.io/).

## Worker

Workers are based on [hibiken/asynq](https://github.com/hibiken/asynq) and use
Redis (or any of the available alternatives such as Valkey and Redict) as a
message passing interface.

## Scheduler

The scheduler is based on [hibiken/asynq](https://github.com/hibiken/asynq) and
is used to trigger the execution of periodic tasks.

# Code Structure

TODO: Document me

# Database

The persistence layer used by the Gardener Inventory is
[PostgreSQL](https://www.postgresql.org/).

# Database Models

The database models are built on top of [GORM](https://gorm.io/).

The following sections provide details about naming conventions and other hints
to follow when creating a new model.

## Base Model

The [pkg/core/models](./pkg/core/models) package provides base models, which are
meant to be used by other models.

Make sure that you embed the [pkg/core/models.Base](./pkg/core/models) model
into your models, so that we have a consistent models structure.

Example model would look like this.

``` go
package mypkg

import coremodels "github.com/gardener/inventory/pkg/core/models"

// Foo does bar
type Foo struct {
    coremodels.Base
    Foo string
    Bar string
}
```

When creating this data model in the database the field names will be converted
to `snake_case` according to the [GORM
conventions](https://gorm.io/docs/models.html#Conventions).

If you need to customize permissions to fields or provide some additional
attributes to your models, please refer to the [Field-Level
Permissions](https://gorm.io/docs/models.html#Field-Level-Permission) and [Field
Tags](https://gorm.io/docs/models.html#Fields-Tags) sections of the GORM
documentation.

Make sure to check the `Register Your Model` section of this document about
details on how to track and migrate your model.

## Table Names

Table names by default will be converted into `snake_case` when you define a new
model.

In order to avoid any conflicts between different package models, each table in
the database should be prefixed with the respective parent package of the model.

For instance, if we have the following Go package and model defined in `pkg/foo/models/models.go`.

``` go
// file: pkg/foo/models/models.go

package models

type Bar struct {
    // struct fields omitted for simplicity
}
```

We should also, prefix the table, which GORM will create with the parent
package, which is `foo` in this example.

In order to do that we need to implement the `gorm.io/gorm/schema.Namer`
interface.

``` go
// TableName implements the [gorm.io/gorm/schema.Namer] interface.
func (Bar) TableName() string {
    return "foo_bar"  // use the foo_ prefix for our table
}
```

## Register Your Model

Once you create a new model, you need to register it with the GORM Atlas
Provider, so that the model is properly tracked and migrated.

In order to do that you need to load your model to
[internal/cmd/atlas-loader](./internal/cmd/atlas-loader/main.go).

# Database Migrations

Database migrations are managed by [Atlas](https://atlasgo.io/) and the
[GORM Atlas Provider](https://github.com/ariga/atlas-provider-gorm).

> NOTE: Do not install Atlas by using `go get ...` or `go install ...` as this
> is no longer supported by the Atlas team.  See this issue for more details on
> this: [ariga/atlas issue #2669](https://github.com/ariga/atlas/issues/2659)

The Atlas configuration for the Gardener Inventory project resides in the
[atlas.hcl](./atlas.hcl) file.

Make sure to check the
[Atlas Getting Started](https://atlasgo.io/getting-started) guide for more
information about Atlas and how to use it.

Other Atlas related documentation worth checking out is:

- [Atlas Project Configuration](https://atlasgo.io/atlas-schema/projects)
- [Atlas URLs](https://atlasgo.io/concepts/url)
- [Atlas Dev Database](https://atlasgo.io/concepts/dev-database)

See how to configure the database URL: https://atlasgo.io/concepts/url

## Migrations Status

Check the database migration status by executing the following command.

``` shell
atlas migrate status --env local
```

## Apply Pending Migrations

Apply any pending migrations by executing the following command.

``` shell
atlas migrate apply --env local
```

## Create New Migration

In order to create a new migration execute the following command, which will
generate a new migration file in the [migrations directory](./migrations).

``` shell
atlas migrate diff --env local <description-of-my-change>
```

# Local Environment

TODO: Document me

# Querying the database

TODO: Document me
