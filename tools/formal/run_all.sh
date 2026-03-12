#!/usr/bin/env bash
set -euo pipefail

jar_path="${1:?TLA+ jar path is required}"
controllers_root="${2:?controllers root is required}"

status=0

for controller_dir in "${controllers_root}"/*; do
  [[ -d "${controller_dir}" ]] || continue
  echo "==> TLC: $(basename "${controller_dir}")"
  if ! "$(dirname "$0")/run_controller.sh" "${jar_path}" "${controller_dir}"; then
    status=1
  fi
done

exit "${status}"
