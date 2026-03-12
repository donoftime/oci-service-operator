#!/usr/bin/env bash
set -euo pipefail

jar_path="${1:?TLA+ jar path is required}"
controller_dir="${2:?controller directory is required}"
spec_path="${controller_dir}/spec.tla"

if [[ ! -f "${spec_path}" ]]; then
  echo "missing spec file for ${controller_dir}" >&2
  exit 1
fi

"$(dirname "$0")/run_tlc.sh" "${jar_path}" "${spec_path}"
