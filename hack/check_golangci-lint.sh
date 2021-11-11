#!/bin/bash
# SPDX-License-Identifier: MIT

set -e

GOLANGCILINT_VERSION="1.43.0"
GOLANGCILINT_FILENAME="golangci-lint-${GOLANGCILINT_VERSION}-linux-amd64.tar.gz"
GOLANGCILINT_URL="https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCILINT_VERSION}/${GOLANGCILINT_FILENAME}"

TMP_BIN="$(pwd)/tmp/bin"
export PATH="${PATH}:${TMP_BIN}"

if ! [ -x "$(command -v golangci-lint-${GOLANGCILINT_VERSION})" ]; then
  # pushd $(mktemp -d)
  echo '[golangci-lint/prepare]: golangci-lint is not installed. Downloading to tmp/bin' >&2

  wget -q "${GOLANGCILINT_URL}"
  tar -xf "${GOLANGCILINT_FILENAME}"
 
  mkdir -p "${TMP_BIN}"
  mv "${GOLANGCILINT_FILENAME%.tar.gz}/golangci-lint" "${TMP_BIN}/golangci-lint-${GOLANGCILINT_VERSION}"
 
  rm -rf "${GOLANGCILINT_FILENAME%.tar.gz}" "${GOLANGCILINT_FILENAME}"
  # popd
fi

echo '[golangci-lint/prepare]: running golangci-lint' >&2
golangci-lint-${GOLANGCILINT_VERSION} run --skip-dirs ./vendor