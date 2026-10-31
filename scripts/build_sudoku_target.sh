#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

GOOS="${GOOS:-$(go env GOOS)}"
GOARCH="${GOARCH:-$(go env GOARCH)}"
PLATFORM_DIR="${GOOS}-${GOARCH}"
OUT_DIR="${OUT_DIR:-${ROOT_DIR}/runtime/bin/${PLATFORM_DIR}}"
SUDOKU_REPO="${SUDOKU_REPO:-https://github.com/SUDOKU-ASCII/sudoku.git}"
SUDOKU_REF="${SUDOKU_REF:-v0.5.2-1}"
PATCH_DIR="${PATCH_DIR:-${ROOT_DIR}/scripts/sudoku_patches}"

mkdir -p "$OUT_DIR"

out="$OUT_DIR/sudoku"
if [[ "$GOOS" == "windows" ]]; then
  out+=".exe"
fi

tmpdir="$(mktemp -d)"
cleanup() { rm -rf "$tmpdir"; }
trap cleanup EXIT

SUDOKU_DIR="${tmpdir}/sudoku"

is_commit_ref() {
  [[ "$1" =~ ^[0-9a-fA-F]{7,40}$ ]]
}

echo "[fetch] sudoku ${SUDOKU_REF} (${SUDOKU_REPO})"
if command -v git >/dev/null 2>&1; then
  cloned=0
  if is_commit_ref "${SUDOKU_REF}"; then
    if git clone --depth 1 "${SUDOKU_REPO}" "${SUDOKU_DIR}" && (
      cd "${SUDOKU_DIR}"
      git checkout --detach "${SUDOKU_REF}" >/dev/null 2>&1 || {
        git fetch --depth 1 origin "${SUDOKU_REF}" >/dev/null 2>&1 &&
        git checkout --detach FETCH_HEAD >/dev/null 2>&1
      }
    ); then
      cloned=1
    fi
  elif git clone --depth 1 --branch "${SUDOKU_REF}" "${SUDOKU_REPO}" "${SUDOKU_DIR}"; then
    cloned=1
  fi

  if [[ "${cloned}" -ne 1 ]]; then
    echo "[warn] git fetch failed; falling back to tarball download"
    if git clone --depth 1 "${SUDOKU_REPO}" "${SUDOKU_DIR}" && (
      cd "${SUDOKU_DIR}"
      git checkout --detach "${SUDOKU_REF}" >/dev/null 2>&1 || {
        git fetch --depth 1 origin "${SUDOKU_REF}" >/dev/null 2>&1 &&
        git checkout --detach FETCH_HEAD >/dev/null 2>&1
      }
    ); then
      :
    else
      rm -rf "${SUDOKU_DIR}"
      mkdir -p "${SUDOKU_DIR}"
      curl -fsSL "https://codeload.github.com/SUDOKU-ASCII/sudoku/tar.gz/${SUDOKU_REF}" \
        | tar -xz -C "${SUDOKU_DIR}" --strip-components=1
    fi
  fi
else
  echo "[warn] git not found; downloading tarball"
  mkdir -p "${SUDOKU_DIR}"
  curl -fsSL "https://codeload.github.com/SUDOKU-ASCII/sudoku/tar.gz/${SUDOKU_REF}" \
    | tar -xz -C "${SUDOKU_DIR}" --strip-components=1
fi

python3 "${ROOT_DIR}/scripts/patch-sudoku.py" "${SUDOKU_DIR}"

echo "[build] sudoku ${GOOS}/${GOARCH} (ref=${SUDOKU_REF}) -> ${out}"
(
  cd "${SUDOKU_DIR}"
  CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
    go build -mod=mod -tags sudoku_patch -trimpath -ldflags "-s -w" \
    -o "$out" \
    ./cmd/sudoku-tunnel
)

echo "[ok] sudoku ready at ${out}"
