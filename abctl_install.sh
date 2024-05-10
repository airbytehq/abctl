#!/bin/bash

set -Eeu

VERSION=0.1

# Logging
# Automatically logs everything that goes to stdout into a file 
# at the same time
DIR_TMP=${DIR_TMP:-$(mktemp -d -t "abctl_install.XXXX")}
exec > >(tee "$DIR_TMP/output.log") 2>&1

# Debug
DEBUG=${DEBUG:-""}
if ! [ -z "$DEBUG" ]; then
    echo "Running in debug mode"
    set -x
fi

# Trap config
_trap() {
    local rv=$?
    if [ "$rv" -ne 0 ]; then
        [ -z "$TELEMETRY_MSG" ] && TELEMETRY_MSG=$(tail -n 1 "$DIR_TMP/output.log")
        _event abctl_install failed "$TELEMETRY_MSG"
        echo
        echo "abctl install failed: $TELEMETRY_MSG"
    else
        _event abctl_install succeeded "$TELEMETRY_MSG"
        echo
        echo "abctl install succeeded. Run:"
        echo
        echo "   abctl --help"
        echo
        echo "to get started! (You might need to sudo, depending on your docker setup)"
    fi
    exit "$rv"
}
trap "_trap" EXIT

# Dev
FORCE_OS=${FORCE_OS:-""}
FORCE_FUN=${FORCE_FUN:-""}

# Defaults
TELEMETRY_ENABLED=${TELEMETRY_ENABLED:-1}
TELEMETRY_STORE=${TELEMETRY_STORE:-~/.airbyte/analytics.yml}
TELEMETRY_KEY=${TELEMETRY_KEY:-"kpYsVGLgxEqD5OuSZAQ9zWmdgBlyiaej"}
TELEMETRY_INSTANCE_ID=${TELEMETRY_INSTANCE_ID:-""}
TELEMETRY_SESSION_ID=${TELEMETRY_SESSION_ID:-""}
TELEMETRY_LOG=
TELEMETRY_MSG=

RELEASE_TAG=${RELEASE_TAG:-""}

DIR_INSTALL=${DIR_INSTALL:-/usr/local/bin}

# Consts

# Helpers
_error() {
    local rv=$?
    
    TELEMETRY_MSG="$1"
    echo "$@" 1>&2
    
    exit "$rv"
}

_sudo() {
    if [ "$(whoami)" = "root" ]; then
        "$@"
    elif _exists sudo; then
        sudo -E "$@"
    else
        _error "Neither root or sudo" ; 
    fi
}

_exists() {
    which "$1" 2>&1 > /dev/null
}

_curl() {
    local url=$1
    local data=${2:-""}

    if _exists curl; then
        if [ -z "$data" ]; then
            curl -Lsf1 --output - "$url"
        else
            curl -Lsf1 --output - "$url" -d "$data" -X POST -H "content-type: application/json"
        fi
    elif _exists wget; then
        if [ -z "$data" ]; then
            wget -q -O - "$url"
        else
            wget -q -O - -H "$url" --post-data "$data" --header "content-type: application/json"
        fi
    else
        _error "Neither curl or wget available." ; 
    fi
}

_extract_value() {
    grep "$1" | cut -d : -f2- | tr -d '"[:space:],'
}

_unique_id() {
    # looks like a ulid but the time piece is in ascii instead of 48 bits encoded
    local time="$(date +"%s000" | head -c 10)"
    local rand="$(LC_ALL=C tr -dc 0123456789ABCDEFGHJKMNPQRSTVWXYZ </dev/urandom | head -c 16)"
    echo "${time}${rand}" | head -c 26
}

_init_telemetry() {
    [ "$TELEMETRY_ENABLED" -eq 1 ] || return 0

    [ -z "$TELEMETRY_SESSION_ID" ] && TELEMETRY_SESSION_ID=$(_unique_id)

    [ -f "$TELEMETRY_STORE" ] && TELEMETRY_INSTANCE_ID=$(< "$TELEMETRY_STORE" _extract_value anonymous_user_id)

    if [ -z "$TELEMETRY_INSTANCE_ID" ]; then
        TELEMETRY_INSTANCE_ID=$(_unique_id)
        
        mkdir -p ~/.airbyte
        echo "# This file is used by Airbyte to track anonymous usage statistics." > "$TELEMETRY_STORE"
        echo "# For more information or to opt out, please see" >> "$TELEMETRY_STORE"
        echo "# - https://docs.airbyte.com/operator-guides/telemetry" >> "$TELEMETRY_STORE"
        echo "anonymous_user_id: $TELEMETRY_INSTANCE_ID" >> "$TELEMETRY_STORE"
    fi
}

_event() {
    local event=$1
    local state=$2
    local message=${3:-""}
    
    [ -z "${TELEMETRY_ENABLED}" ] && return 0

    # ensure we don't log the same event twice
    echo "$TELEMETRY_LOG" | grep -q "$event-$state" && return 0

    local telemetry_request=$(cat <<EOF
{
    "anonymousId":"$TELEMETRY_INSTANCE_ID",
    "event":"$event",
    "properties": {
        "session_id":"$TELEMETRY_SESSION_ID",
        "state":"$state",
        "os":"$(_get_os)",
        "script_version":"$VERSION",
        "error":"$message"
    },
    "timestamp":"$(date -u "+%Y-%m-%dT%H:%M:%SZ")",
    "writeKey":"$TELEMETRY_KEY"
}
EOF
)

    _curl "https://api.segment.io/v1/track" "$telemetry_request" > /dev/null
    TELEMETRY_LOG="$event-$state-$TELEMETRY_LOG"
}

_install_binary() {
    local os="$1"
    local arch="$2"

    local url_fragment=releases/latest
    [ -z "$RELEASE_TAG" ] || url_fragment="releases/tags/$RELEASE_TAG"

    local release_data=$(_curl "https://api.github.com/repos/airbytehq/abctl/$url_fragment")
    local release_tag=$(echo "$release_data" | _extract_value tag_name)
    local release_url=$(echo "$release_data" | grep "$os-$arch" | _extract_value browser_download_url)
    local release_filename=$(echo "$release_data" | grep "$os-$arch" | _extract_value name)
    
    echo "Downloading abctl from $release_url"

    _curl "$release_url" > "${DIR_TMP}/$release_filename"

    (
        cd "${DIR_TMP}"
        mkdir -p release
        tar zxf "${release_filename}" -C release
        local binary=$(ls -1 release/*/abctl | head -n 1)
        echo "Installing '$binary' to ${DIR_INSTALL}"
        chmod +x "$binary"
        _sudo cp "$binary" "${DIR_INSTALL}"
    )
}

_get_os() {
    if ! [ -z "${FORCE_OS}" ]; then 
        echo "${FORCE_OS}"
    elif [ "$(uname)" = "Linux" ]; then
        echo linux
    elif [ "$(uname)" = "Darwin" ]; then
        echo darwin
    elif uname -r | grep -c Microsoft; then
        echo "windows"
    else
        _error "Unknown system."
    fi
}

_get_arch() {
    if uname -m | grep -q "arm"; then
        echo arm64
    else
        echo amd64
    fi
}

# System installs
_install_linux() {
    echo "Installing for Linux..."

    # We should probably handle yum & deb packages instead of installing the binary directly
    _install_binary linux "$(_get_arch)"
}

_install_darwin() {
    echo "Installing for Darwin..."

    if ! _exists brew; then
        _install_binary darwin "$(_get_arch)"
    elif brew ls --version abctl > /dev/null; then
        brew upgrade abctl
    else
        brew tap airbytehq/tap
        brew install abctl 
    fi
}

_install_windows() {
    echo "Installing for Windows..."
    echo "Unsupported"
}

_test_binary() {
    abctl --help > /dev/null
}

main() {
    [ -z "${FORCE_FUN}" ] || { "$@"; exit 0; }

    _init_telemetry

    _event abctl_install started

    "_install_$(_get_os)" "$@"

    _test_binary
}

main "$@"
