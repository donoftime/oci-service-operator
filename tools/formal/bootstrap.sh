#!/usr/bin/env bash
set -euo pipefail

tool_dir="${1:?tool directory is required}"
tla_version="${2:?TLA+ version is required}"
plantuml_version="${3:?PlantUML version is required}"

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
tla_jar="${tool_dir}/tla2tools-${tla_version}.jar"
plantuml_jar="${tool_dir}/plantuml-${plantuml_version}.jar"

tla_url="${TLA2TOOLS_URL:-https://github.com/tlaplus/tlaplus/releases/download/v${tla_version}/tla2tools.jar}"
plantuml_url="${PLANTUML_URL:-https://repo1.maven.org/maven2/net/sourceforge/plantuml/plantuml/${plantuml_version}/plantuml-${plantuml_version}.jar}"

mkdir -p "${tool_dir}"

copy_first_existing() {
  local dest="$1"
  shift

  if [[ -f "${dest}" ]]; then
    return 0
  fi

  for candidate in "$@"; do
    if [[ -n "${candidate}" && -f "${candidate}" ]]; then
      cp "${candidate}" "${dest}"
      return 0
    fi
  done

  return 1
}

download_if_missing() {
  local url="$1"
  local dest="$2"
  local tmp

  if [[ -f "${dest}" ]]; then
    return
  fi

  tmp="$(mktemp "${dest}.tmp.XXXXXX")"
  trap 'rm -f "${tmp}"' EXIT
  curl -fsSL "${url}" -o "${tmp}"
  mv "${tmp}" "${dest}"
  trap - EXIT
}

install_plantuml() {
  local candidates=()

  if [[ -n "${PLANTUML_JAR:-}" ]]; then
    candidates+=("${PLANTUML_JAR}")
  fi
  while IFS= read -r candidate; do
    candidates+=("${candidate}")
  done < <(find "${HOME}/.vscode/extensions" -maxdepth 2 -name 'plantuml.jar' 2>/dev/null | sort)

  if ! copy_first_existing "${plantuml_jar}" "${candidates[@]}"; then
    download_if_missing "${plantuml_url}" "${plantuml_jar}"
  fi
}

if ! copy_first_existing "${tla_jar}" "${TLA2TOOLS_JAR:-}" "${repo_root}/bin/tla2tools.jar"; then
  download_if_missing "${tla_url}" "${tla_jar}"
fi

install_plantuml
