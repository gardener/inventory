# Database

The persistence layer used by the Gardener Inventory is
[PostgreSQL](https://www.postgresql.org/).

# Data Model

The high-level overview of the data model followed by the Gardener Inventory
system looks like this.

![Data Model](../images/data-model.png)

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
