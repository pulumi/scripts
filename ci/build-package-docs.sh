#!/bin/bash
# build-package-docs.sh updates the docs generated from the indicated repo
set -o nounset
set -o errexit
set -o pipefail

if [[ "${TRAVIS:-}" != "true" ]]; then
	echo "error: this script should be run from within Travis"
	exit 1
fi

if [[ -z "${1:-}" ]]; then
	echo "usage $0 <package-simple-name>"
	exit 1
fi

if ! echo "${TRAVIS_TAG:-}" | grep -q -e "^v[0-9]\+\.[0-9]\+\.[0-9]\+$"; then
	echo "Skipping documentation generation; ${TRAVIS_TAG:-} does not denote a released version"
	exit 0
fi

PKG_NAME="$1"
VERSION=${TRAVIS_TAG#"v"}

echo "Building SDK docs for version ${VERSION}:"

# Clone the docs repo and fetch its dependencies.
git clone "https://github.com/pulumi/docs.git" "$(go env GOPATH)/src/github.com/pulumi/docs"
cd "$(go env GOPATH)/src/github.com/pulumi/docs"
make ensure

go get -u github.com/cbroglie/mustache
go get -u github.com/gobuffalo/packr
go get -u github.com/pkg/errors

# Regenerate the Node.JS SDK docs
PKGS=${PKG_NAME} NOBUILD=true ./scripts/run_typedoc.sh

# Regenerate the Python docs
./scripts/generate_python_docs.sh

if [[ "${PKG_NAME}" == "pulumi" ]]; then
	# Regenerate the CLI docs
	pulumi gen-markdown ./content/docs/reference/cli

	# Update latest-version
	echo -n "${VERSION}" > ./static/latest-version

	# Update the version list
	NL=$'\n' 
	sed -e "s/<tbody>/<tbody>\\${NL}        {{< changelog-table-row version=\"${VERSION}\" date=\"$(date +%Y-%m-%d)\" >}}/" -i ./content/docs/get-started/install/versions.md
fi

# Commit the resulting changes
BRANCH="${PKG_NAME}/${TRAVIS_JOB_NUMBER}"
MESSAGE="Regen docs for ${PKG_NAME}@${VERSION}"

git checkout -b "${BRANCH}"
git config user.name "Pulumi Bot"
git config user.email "bot@pulumi.com"
git add .
git commit --allow-empty -m "${MESSAGE}"

# If we have a token for pulumi-bot, push up the changes and add a status
# to a github compare.
if [ ! -z "${PULUMI_BOT_GITHUB_API_TOKEN:-}" ]; then
	# Push up the resulting changes
	git remote add pulumi-bot "https://pulumi-bot:${PULUMI_BOT_GITHUB_API_TOKEN}@github.com/pulumi-bot/docs"
	git push pulumi-bot --set-upstream --force "${BRANCH}"

	# Create a pull request in the docs repo.
	BODY="{\"title\": \"${MESSAGE}\", \"head\": \"pulumi-bot:${BRANCH}\", \"base\": \"master\"}"
	curl -u "pulumi-bot:${PULUMI_BOT_GITHUB_API_TOKEN}" -X POST -H "Content-Type: application/json" -d "${BODY}" "https://api.github.com/repos/pulumi/docs/pulls"
else
	# Otherwise, just print out the diff to the build log.
	git diff HEAD~1 HEAD
fi

exit 0
