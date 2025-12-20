#!/usr/bin/env bash
set -euo pipefail

BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_DIR="${BASE_DIR}/logs"
mkdir -p "${LOG_DIR}"

CONFIG_PATH="${CONFIG_PATH:-${BASE_DIR}/config.json}"
INTERVAL_SECONDS="${INTERVAL_SECONDS:-1800}"

BIN_PATH="${BIN_PATH:-${BASE_DIR}/p2a}"
if [[ ! -x "${BIN_PATH}" ]]; then
  (cd "${BASE_DIR}" && go build -o "${BIN_PATH}" ./cmd/p2a)
fi

while true; do
  DATE_STR="$(date +%F)"
  TS_STR="$(date -u +%FT%TZ)"
  LOG_FILE="${LOG_DIR}/p2a-${DATE_STR}.log"

  {
    echo "----- ${TS_STR} sync start -----"
    "${BIN_PATH}" -config "${CONFIG_PATH}"
    echo "----- ${TS_STR} sync end (exit=0) -----"
  } >> "${LOG_FILE}" 2>&1 || {
    EXIT_CODE=$?
    echo "----- ${TS_STR} sync end (exit=${EXIT_CODE}) -----" >> "${LOG_FILE}" 2>&1
  }

  sleep "${INTERVAL_SECONDS}"
done
