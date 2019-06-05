#!/usr/bin/env bash

set -o errexit
set -o pipefail

# Note the ordering here is designed to prevent problems hitting the "big 3"
# providers by not doing them first.
PROVIDERS="digitalocean packet newrelic cloudflare linode f5bigip newrelic "
PROVIDERS+="random vsphere openstack gcp azure aws"

CHANGELOG_ENTRY=
while getopts ":m:" arg; do
  case "${arg}" in
    m)
      CHANGELOG_ENTRY=$OPTARG
      ;;
  esac
done

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

update_dependency() {
	local repoPath=$1
	local upstreamImportPath=$2
	local upstreamTag=$3

	bash <<-EOF
	cd "${repoPath}"
	GO111MODULE=on go get "${upstreamImportPath}@${upstreamTag}"
	EOF
}

ensure_vendor() {
	local repoPath=$1

	bash <<-EOF
	cd "${repoPath}"
	make ensure
	EOF
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

build_repo() {
	local repoPath=$1

	bash <<-EOF
	cd "${repoPath}"
	make
	EOF
}

clone_repo_if_not_exists() {
	local repoPath=$1
	local repoName=$2

	if [ ! -d "${repoPath}" ] ; then
		git clone "git@github.com:pulumi/pulumi-${repoName}"
	fi
}

push_and_pull_request() {
	local repoPath=$1
	local branchName=$2
	local depName=$3
	local depRef=$4

	bash <<-EOF
	cd "${repoPath}"

	git push origin "${branchName}"

	hub pull-request \
		--base master \
		--head "${branchName}" \
		--message "Update "${depName}" to ${depRef}" \
		--message "This PR updates \\\`${depName}\\\` to ${depRef}, and re-runs code generation" \
		--reviewer "stack72, jen20" \
		--labels "area/providers"
	EOF
}

add_changelog() {
	local repoPath=$1
	local changelogNote=$2

	bash <<-EOF
	cd "${repoPath}"

	chg add "${changelogNote}"
	EOF
}

find_latest_sha() {
	local repoOwnerAndName=$1
	local branchName=$2

	curl --silent -L \
		"https://api.github.com/repos/${repoOwnerAndName}/branches/${branchName}" | \
		jq -M -r '.commit.sha'
}

PTF_IMPORT_PATH="github.com/pulumi/pulumi-terraform"
PTF_TAG="master"
PTF_SHA=$(find_latest_sha "pulumi/pulumi-terraform" "master")

for PROVIDER_SUFFIX in ${PROVIDERS}
do
	PROVIDER_REPO="pulumi-${PROVIDER_SUFFIX}"
	PROVIDER_REPO_PATH="$(go env GOPATH)/src/github.com/pulumi/${PROVIDER_REPO}"
	BRANCH_NAME=deps/update-pulumi-terraform-${PTF_SHA:0:10}

	echo "Updating pulumi-terraform in ${PROVIDER_REPO}..."
	clone_repo_if_not_exists "${PROVIDER_REPO_PATH}" "${PROVIDER_REPO}"
	make_clean_worktree "${PROVIDER_REPO_PATH}" "${BRANCH_NAME}"
	update_dependency "${PROVIDER_REPO_PATH}" "${PTF_IMPORT_PATH}" "${PTF_TAG}"
	ensure_vendor "${PROVIDER_REPO_PATH}"
	commit_changes "${PROVIDER_REPO_PATH}" "Update go.{mod,sum} for latest pulumi-terraform"
	build_repo "${PROVIDER_REPO_PATH}"
	commit_changes "${PROVIDER_REPO_PATH}" "Run tfgen with latest pulumi-terraform"

	if [ -n "${CHANGELOG_ENTRY}" ] ; then
		add_changelog "${PROVIDER_REPO_PATH}" "${CHANGELOG_ENTRY} ([${PTF_SHA:0:10}](https://${PTF_IMPORT_PATH}/commit/${PTF_SHA}))"
		commit_changes "${PROVIDER_REPO_PATH}" "Update CHANGELOG.md"
	fi

	push_and_pull_request "${PROVIDER_REPO_PATH}" "${BRANCH_NAME}" "pulumi-terraform" "${PTF_SHA:0:10}"
done
