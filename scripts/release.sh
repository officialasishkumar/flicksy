#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-snapshot}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
BINARY_NAME="flicksy"
MAIN_PACKAGE="./cmd/flicksy"
GO_BIN="${GO:-go}"

TARGETS=(
  "linux amd64 tar.gz"
  "linux arm64 tar.gz"
  "darwin amd64 tar.gz"
  "darwin arm64 tar.gz"
  "windows amd64 zip"
)

checksum_file() {
  local file_path="$1"

  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file_path"
    return
  fi

  shasum -a 256 "$file_path"
}

rm -rf "${DIST_DIR}"
mkdir -p "${DIST_DIR}"

for target in "${TARGETS[@]}"; do
  read -r goos goarch archive_format <<<"${target}"

  archive_basename="${BINARY_NAME}_${VERSION}_${goos}_${goarch}"
  staging_dir="$(mktemp -d)"
  binary_filename="${BINARY_NAME}"

  if [[ "${goos}" == "windows" ]]; then
    binary_filename="${BINARY_NAME}.exe"
  fi

  GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=0 "${GO_BIN}" build -trimpath -o "${staging_dir}/${binary_filename}" "${MAIN_PACKAGE}"
  cp "${ROOT_DIR}/README.md" "${staging_dir}/README.md"
  cp "${ROOT_DIR}/.env.example" "${staging_dir}/.env.example"

  if [[ "${archive_format}" == "tar.gz" ]]; then
    tar -C "${staging_dir}" -czf "${DIST_DIR}/${archive_basename}.tar.gz" .
  else
    (
      cd "${staging_dir}"
      zip -q -r "${DIST_DIR}/${archive_basename}.zip" .
    )
  fi

  rm -rf "${staging_dir}"
done

(
  cd "${DIST_DIR}"
  checksum_output="$(mktemp)"
  for artifact in *; do
    checksum_file "${artifact}"
  done > "${checksum_output}"
  mv "${checksum_output}" checksums.txt
)
