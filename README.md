# inventory

The Gardener Inventory is a system, which collects resources from various data
sources, persists the data, and establishes relationships between the resources.

The collected data can be later analyzed to show the relationship and
dependencies between the various resources.

# Requirements

- Go 1.22.x or later
- [Redis](https://redis.io/)
- [PostgreSQL](https://www.postgresql.org/)

[Valkey](https://github.com/valkey-io/valkey) or [Redict](https://redict.io),
can be used instead of Redis.

Additional requirements for local development.

- [Docker](https://www.docker.com/)
- [Docker Compose](https://docs.docker.com/compose/)
- [minikube](https://minikube.sigs.k8s.io/docs/drivers/)

# Documentation

- [Design Goals](./docs/design.md)
- [Development Guide](./docs/development.md)
- [Operations Guide](./docs/ops-guide.md)
- [Database Queries](./docs/db-queries.md)
- [AWS OIDC Trust](./docs/oidc-aws.md)
- [GCP OIDC Trust](./docs/oidc-gcp.md)
- [Azure OIDC Trust](./docs/oidc-azure.md)
- [Deployment](./deployment)

# Contributing

Gardener Inventory is hosted on [Github](https://github.com/gardener/inventory).

Please contribute by reporting issues, suggesting features or by sending patches
using pull requests.

# License

This project is Open Source and licensed under [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0).
