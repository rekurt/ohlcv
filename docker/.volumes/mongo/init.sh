#!/bin/bash
set -e

envsubst < /docker-entrypoint-initdb.d/init_template.js > /init.js
mongosh -u "$MONGO_INITDB_ROOT_USERNAME" -p "$MONGO_INITDB_ROOT_PASSWORD" -f /init.js
