#!/bin/env bash

readonly config_dir=config
readonly docker_dir=${DOCKER_DIR-docker}

config=$config_dir/.env
if [ ! -f $config_dir/.env ]; then
  config=$config_dir/example.env
  echo "[WARNING!] Using $config default config!"
fi

# shellcheck disable=SC2046
export $(grep -v '^#' "$config" | xargs -0)


services=$(echo "$@" | grep -Eo '([a-z][a-z0-9_\-]+)+\.' | sed 's/.//')
docker_compose_files_args=()
for svc in $services
do
  docker_compose_files_args+=('-f' "${docker_dir}/${svc}.docker-compose.yml")
done

ARGS=( "$@" ); ARGS_AFTER_FIRST=( "${ARGS[@]:1}" )
echo "${docker_compose_files_args[@]}" "${ARGS[@]:1}"

docker-compose \
  --project-directory "$(pwd)"\
  --env-file "${CFG_FILE}" \
  "${docker_compose_files_args[@]}" \
  "${ARGS[@]:1}"
