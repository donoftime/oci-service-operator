#!/usr/bin/env bash
set -euo pipefail

jar_path="${1:?PlantUML jar path is required}"
controller_dir="${2:?controller directory is required}"
diagram_dir="${controller_dir}/diagrams"

if [[ ! -d "${diagram_dir}" ]]; then
  exit 0
fi

shopt -s nullglob
diagram_files=("${diagram_dir}"/*.puml)
shopt -u nullglob

if [[ "${#diagram_files[@]}" -eq 0 ]]; then
  exit 0
fi

java -Djava.awt.headless=true -jar "${jar_path}" -tsvg "${diagram_files[@]}"
