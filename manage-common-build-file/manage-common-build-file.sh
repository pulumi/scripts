#!/usr/bin/env bash

set -o errexit
set -o pipefail

# Note the ordering here is designed to prevent problems hitting the "big 3"
# providers by not doing them first.
PROVIDERS="digitalocean packet newrelic cloudflare linode f5bigip newrelic "
PROVIDERS+="gitlab mysql postgresql datadog dnsimple "
PROVIDERS+="random vsphere openstack gcp azure azuread aws"

clone_repo_if_not_exists() {
	local repoPath=$1
	local repoName=$2

	if [ ! -d "${repoPath}" ] ; then
		git clone "git@github.com:pulumi/pulumi-${repoName}"
	fi
}

make_clean_worktree() {
	local repoPath=$1
	local branchName=$2

	dirtyStatus="$(cd "${repoPath}" && git status -s)"

	bash  <<-EOF
	echo "${dirtyStatus}"

	cd "${repoPath}"
	if [ -n "${dirtyStatus}" ] ; then
		git add .
		git stash save --all "Stash changes before updating provider"
	fi

	git checkout master
	git pull origin master
	git checkout -b "${branchName}"
	EOF
}

manage_common_build_file() {
    local repoPath=$1

    bash <<-EOF
    cp -f ./common.mk ${repoPath}/build/common.mk
EOF
}

find_latest_sha() {
	local repoOwnerAndName=$1
	local branchName=$2

	curl --silent -L \
		"https://api.github.com/repos/${repoOwnerAndName}/branches/${branchName}" | \
		jq -M -r '.commit.sha'
}

commit_changes() {
	local repoPath=$1
	local commitMessage=$2

	bash <<-EOF
	cd "${repoPath}"

	git add .
	git commit -a -m "${commitMessage}"
EOF
}

push_and_pull_request() {
	local repoPath=$1
	local branchName=$2
	local depRef=$3

	bash <<-EOF
	cd "${repoPath}"

	git push origin "${branchName}"

	hub pull-request \
		--base master \
		--head "${branchName}" \
		--message "Update .travis.yml to ${depRef:0:10}" \
		--message "This PR updates build/common.mk to [${depRef:0:10}](https://github.com/pulumi/scripts/commit/${depRef})" \
		--reviewer "stack72,jen20" \
		--labels "area/providers,impact/no-changelog-required"
EOF
}

PS_SHA=$(find_latest_sha "pulumi/scripts" "master")

for PROVIDER_SUFFIX in ${PROVIDERS}
do
  PROVIDER_REPO="pulumi-${PROVIDER_SUFFIX}"
  PROVIDER_REPO_PATH="$(go env GOPATH)/src/github.com/pulumi/${PROVIDER_REPO}"
  BRANCH_NAME=build/update-common-build-${PS_SHA:0:10}

  echo "Updating build/common.mk in ${PROVIDER_REPO}..."
  clone_repo_if_not_exists "${PROVIDER_REPO_PATH}" "${PROVIDER_REPO}"
  make_clean_worktree "${PROVIDER_REPO_PATH}" "${BRANCH_NAME}"
  manage_common_build_file "${PROVIDER_REPO_PATH}"
  commit_changes "${PROVIDER_REPO_PATH}" "Update build/common.mk"

  push_and_pull_request "${PROVIDER_REPO_PATH}" "${BRANCH_NAME}" "${PS_SHA}"
done
