#!/bin/sh
set -eu

printf '%s:5432:*:replicator:%s\n' "$PRIMARY_HOST" "$REPLICATION_PASSWORD" > /var/lib/postgresql/.pgpass
chmod 0600 /var/lib/postgresql/.pgpass
export PGPASSFILE=/var/lib/postgresql/.pgpass

if [ ! -s "$PGDATA/PG_VERSION" ]; then
  rm -rf "${PGDATA:?}"/*
  until pg_basebackup \
    -h "$PRIMARY_HOST" -U replicator -D "$PGDATA" -Fp -Xs -P -R
  do
    echo "primary is not ready for base backup; retrying"
    sleep 2
  done
  chmod 0700 "$PGDATA"
fi

exec postgres -c hot_standby=on
