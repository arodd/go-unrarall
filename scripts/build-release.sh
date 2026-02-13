#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <version>" >&2
  echo "example: $0 1.0.0" >&2
  exit 1
fi

raw_version="$1"
version="${raw_version#v}"
tag="v${version}"

if [[ -z "${version}" ]]; then
  echo "version must not be empty" >&2
  exit 1
fi

for tool in go tar zip sha256sum; do
  if ! command -v "${tool}" >/dev/null 2>&1; then
    echo "required tool not found: ${tool}" >&2
    exit 1
  fi
done

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist_dir="${root_dir}/dist/${tag}"
staging_dir="${dist_dir}/staging"

rm -rf "${dist_dir}"
mkdir -p "${staging_dir}"

targets=(
  "linux amd64 tar.gz"
  "linux arm64 tar.gz"
  "darwin amd64 tar.gz"
  "darwin arm64 tar.gz"
  "windows amd64 zip"
)

for target in "${targets[@]}"; do
  read -r goos goarch archive_ext <<<"${target}"
  artifact_base="go-unrarall_${version}_${goos}_${goarch}"
  package_dir="${staging_dir}/${artifact_base}"
  binary_name="unrarall"

  if [[ "${goos}" == "windows" ]]; then
    binary_name="unrarall.exe"
  fi

  mkdir -p "${package_dir}"

  CGO_ENABLED=0 GOOS="${goos}" GOARCH="${goarch}" \
    go build -trimpath -ldflags "-s -w -X main.version=${version}" \
    -o "${package_dir}/${binary_name}" ./cmd/unrarall

  cp "${root_dir}/README.md" "${package_dir}/README.md"
  cp "${root_dir}/CHANGELOG.md" "${package_dir}/CHANGELOG.md"

  if [[ "${archive_ext}" == "zip" ]]; then
    (cd "${package_dir}" && zip -q -9 -r "${dist_dir}/${artifact_base}.zip" .)
  else
    (cd "${package_dir}" && tar -czf "${dist_dir}/${artifact_base}.tar.gz" .)
  fi
done

rm -rf "${staging_dir}"

(
  cd "${dist_dir}"
  sha256sum go-unrarall_"${version}"_* > SHA256SUMS
)

cat >"${dist_dir}/RELEASE_NOTES.md" <<EOF
## go-unrarall ${tag}

### Binaries
- \`go-unrarall_${version}_linux_amd64.tar.gz\`
- \`go-unrarall_${version}_linux_arm64.tar.gz\`
- \`go-unrarall_${version}_darwin_amd64.tar.gz\`
- \`go-unrarall_${version}_darwin_arm64.tar.gz\`
- \`go-unrarall_${version}_windows_amd64.zip\`

### Checksums
- \`SHA256SUMS\` contains SHA-256 hashes for every release archive.
EOF

echo "release artifacts ready: ${dist_dir}"
ls -1 "${dist_dir}"
