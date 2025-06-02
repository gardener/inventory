#!/usr/bin/env bash
#
# The following script is used to sync the configuration files used by the
# existing kustomize variants in `deployment/kustomize' with the sample config
# and data files from the repo, e.g. sample inventory config, dashboards, etc.

set -e

_SCRIPT_NAME="${0##*/}"
_SCRIPT_DIR=$( dirname `readlink -f -- "${0}"` )
_REPO_ROOT="${_SCRIPT_DIR}/.."
_KUSTOMIZE_DIR="${_REPO_ROOT}/deployment/kustomize"

# Update sample config
install -m 0644 \
        "${_REPO_ROOT}/examples/config.yaml" \
        "${_KUSTOMIZE_DIR}/config/secrets/config.yaml"

# Update Grafana dashboards
rsync -av --exclude "*~" \
      "${_REPO_ROOT}/extra/grafana/" \
      "${_KUSTOMIZE_DIR}/grafana/config/files"
