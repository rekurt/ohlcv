#!/bin/env bash

readonly config_dir=${CONFIG_DIR:-config}
readonly profiles=${COMPOSE_PROFILES:-db,queue,ws}
readonly docker_dir=${DOCKER_DIR:-docker}

# shellcheck disable=SC2012
compose_configs=$(ls "$docker_dir"/*compose.yml | sed -e s"/$docker_dir/\-f $docker_dir/")

config=$config_dir/.env
if [ ! -f $config_dir/.env ]; then
  config=$config_dir/example.env
  echo "[WARNING!] Using $config default config!"
fi
# shellcheck disable=SC2046
# export $(grep -v '^#' "$config" | xargs -0)
# shellcheck disable=SC1090
$(source "$config")

echo "COMPOSE_PROFILES=$profiles docker-compose \\
--project-directory $(pwd) \\
--env-file $config \\
$(echo "$compose_configs" | xargs)
$@
"

COMPOSE_PROFILES="$profiles" docker-compose \
  --project-directory "$(pwd)" \
  --env-file "$config" \
  $(echo $compose_configs | xargs) "$@"
