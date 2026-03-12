#!/usr/bin/env bash
set -euo pipefail

jar_path="${1:?PlantUML jar path is required}"
controllers_root="${2:?controllers root is required}"

for controller_dir in "${controllers_root}"/*; do
  [[ -d "${controller_dir}" ]] || continue
  echo "==> Diagram: $(basename "${controller_dir}")"
  "$(dirname "$0")/render_controller.sh" "${jar_path}" "${controller_dir}"
done
