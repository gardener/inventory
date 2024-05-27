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

TODO: Document me

# Database

Please refer to the [Database](./docs/database.md) document for more details.

# Local Environment

TODO: Document me

# Querying the database

TODO: Document me
