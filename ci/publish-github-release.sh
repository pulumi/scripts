#!/usr/bin/env bash
# publish-github-release.sh creates a "release" in the github repo and associates with the build tag

# usage publish-github-release.sh REPO VERSION
set -o errexit
set -o pipefail

if [[ -z "${PULUMI_BOT_GITHUB_API_TOKEN:-}" ]]; then
    echo "error: PULUMI_BOT_GITHUB_API_TOKEN must be set"
    exit 1
fi

if [ "$#" -ne 2 ]; then
    >&2 echo "usage: $0 REPO VERSION"
    exit 1
fi

PAYLOAD="{\"tag_name\": \"${2}\", \"name\": \"${2}\"}"

curl \
  -u "pulumi-bot:${PULUMI_BOT_GITHUB_API_TOKEN}" \
  -H "Accept: application/json" \
  -H "Content-Type:application/json" \
  -X POST --data "${PAYLOAD}" "https://api.github.com/repos/pulumi/${1}/releases"
