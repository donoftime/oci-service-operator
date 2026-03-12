#!/usr/bin/env bash
set -euo pipefail

jar_path="${1:?TLA+ jar path is required}"
spec_path="${2:?spec path is required}"

jar_path="$(cd "$(dirname "${jar_path}")" && pwd)/$(basename "${jar_path}")"

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
shared_dir="${repo_root}/formal/shared"
spec_dir="$(dirname "${spec_path}")"
spec_file="$(basename "${spec_path}")"
module_name="${spec_file%.tla}"
cfg_path="${spec_dir}/${module_name}.cfg"
log_dir="${spec_dir}/out"
log_path="${log_dir}/tlc.log"
work_dir="$(mktemp -d "/tmp/osok-tlc-${module_name}.XXXXXX")"

cleanup() {
  rm -rf "${work_dir}"
}
trap cleanup EXIT

if [[ ! -f "${cfg_path}" ]]; then
  echo "missing cfg file for ${spec_path}" >&2
  exit 1
fi

mkdir -p "${log_dir}"

find "${spec_dir}" -maxdepth 1 -type f \( -name '*.tla' -o -name '*.cfg' \) -exec cp {} "${work_dir}/" \;
if [[ -d "${shared_dir}" ]]; then
  find "${shared_dir}" -maxdepth 1 -type f -name '*.tla' -exec cp {} "${work_dir}/" \;
fi

(
  cd "${work_dir}"
  java -cp "${jar_path}" tlc2.TLC -cleanup -dfid 32 -config "$(basename "${cfg_path}")" "${module_name}"
) | tee "${log_path}"
