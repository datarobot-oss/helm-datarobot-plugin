#!/usr/bin/env bash

set -e
set -u

PROJECT_NAME="helm-datarobot-plugin"
PROJECT_GH="datarobot-oss/$PROJECT_NAME"

HELM_PLUGIN_NAME="helm-datarobot"

if [ -n "${SKIP_BIN_INSTALL:-}" ]; then
    echo "Development mode: not downloading versioned release."
    exit 0
fi


validate_checksum() {
    if ! grep -q ${1} ${2}; then
        echo "Invalid checksum" > /dev/stderr
        exit 1
    fi
    echo "Checksum is valid."
}

initArch() {
  ARCH=$(uname -m)
  case $ARCH in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)
      echo "Arch '$(uname -m)' not supported!" >&2
      exit 1
      ;;
  esac

}

initOS() {
  OS=$(uname -s)
  case "$(uname)" in
    Darwin) OS="darwin" ;;
    Linux) OS="linux" ;;
      echo "OS '$(uname)' not supported!" >&2
      exit 1
      ;;
  esac
}


downloadRelease() {
  SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  PLUGIN_VERSION=$(grep -e  '^version' $SCRIPT_DIR/../plugin.yaml | sed -e 's/^version: //')
  echo "Downloading and installing ${HELM_PLUGIN_NAME} ${PLUGIN_VERSION} ..."
  BIN_URL="https://github.com/${PROJECT_GH}/releases/download/${PLUGIN_VERSION}/${HELM_PLUGIN_NAME}-${OS}-${ARCH}"
  CHECKSUM_URL="https://github.com/${PROJECT_GH}/releases/download/${PLUGIN_VERSION}/${HELM_PLUGIN_NAME}_${PLUGIN_VERSION}_checksums.txt"

  PLUGIN_TMP_FILE="/tmp/${HELM_PLUGIN_NAME}-${PLUGIN_VERSION}-${OS}-${ARCH}"
  CHECKSUM_TMP_FILE="/tmp/${HELM_PLUGIN_NAME}_${PLUGIN_VERSION}_checksums.txt"

  echo "Downloading $BIN_URL"
  if type "curl" > /dev/null; then
    curl -L "$BIN_URL" -o "$PLUGIN_TMP_FILE"
    curl -L "$CHECKSUM_URL" -o "$CHECKSUM_TMP_FILE"
  elif type "wget" > /dev/null; then
    wget -q -O "$PLUGIN_TMP_FILE" "$BIN_URL"
    wget -q -O "$CHECKSUM_TMP_FILE" "$CHECKSUM_URL"
  fi

  if command -v sha256sum >/dev/null 2>&1; then
    checksum=$(sha256sum ${PLUGIN_TMP_FILE} | awk '{ print $1 }')
    validate_checksum ${checksum} ${CHECKSUM_TMP_FILE}
  elif command -v openssl >/dev/null 2>&1; then
    checksum=$(openssl dgst -sha256 ${PLUGIN_TMP_FILE} | awk '{ print $2 }')
    validate_checksum ${checksum} ${CHECKSUM_TMP_FILE}
  else
    echo "WARNING: no tool found to verify checksum" > /dev/stderr
  fi
}

# downloadFile downloads the latest binary package and also the checksum
# for that binary.
installBin() {
  echo "Preparing to install into ${HELM_PLUGIN_DIR}"
  mkdir -p "$HELM_PLUGIN_DIR/bin"
  mv $PLUGIN_TMP_FILE "$HELM_PLUGIN_DIR/bin/$PROJECT_NAME"
  chmod +x "$HELM_PLUGIN_DIR/bin/$PROJECT_NAME"
}

# testVersion tests the installed client to make sure it is working.
testVersion() {
  set +e
  echo "$PROJECT_NAME installed into $HELM_PLUGIN_DIR/bin/$PROJECT_NAME"
  "${HELM_PLUGIN_DIR}/bin/$PROJECT_NAME" version
  set -e
}



# fail_trap is executed if an error occurs.
fail_trap() {
  result=$?
  if [ "$result" != "0" ]; then
    echo "Failed to install $PROJECT_NAME"
    echo "\tFor support, go to https://github.com/kubernetes/helm."
  fi
  exit $result
}



# Execution

#Stop execution on any error
trap "fail_trap" EXIT
set -e
initArch
initOS
downloadRelease
installBin
testVersion
