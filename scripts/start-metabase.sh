#!/usr/bin/env bash

set -e

# Create the metabase database, unless already existing
psql -t -c 'SELECT datname FROM pg_catalog.pg_database' | \
  grep metabase >& /dev/null || \
  createdb metabase

docker compose --profile metabase up metabase
