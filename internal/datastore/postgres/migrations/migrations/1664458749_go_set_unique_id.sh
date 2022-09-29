#!/usr/bin/env bash

if [[ -z "${SPICEDB_DATASTORE_CONN_URI}" ]]; then
  echo "SPICEDB_DATASTORE_CONN_URI not set, must provide a datastore URI"
  exit 1
fi

spicedb migrate --datastore-engine=postgres 0010_go_set_unique_id
