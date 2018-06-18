#!/bin/bash
# publish.sh will publish the current bits from the usual build location to an S3 build share.  The first argument is
# the file to publish, the second is the name of the release ("lumi", "aws", etc), and all other arguments are the Git
# commitish(es) to use (commit hash, tag, etc).  Multiple may be passed, for instance if you'd like to use the commit,
# the current branch, and a tag; the script will be smart about uploading as little as possible.

set -o nounset -o errexit -o pipefail

cmderr() {
    echo "error: $1"
    echo "usage: publish.sh FILE RELEASE VERSION..."
    echo "   where FILE is the release package to publish"
    echo "         RELEASE is something like pulumi-fabric, etc."
    echo "         VERSION is one or more commitishes, tags, etc. for this release"
    exit 1
}

PUBFILE=${1}
if [ -z "${PUBFILE}" ]; then
    cmderr "missing the file to publish"
fi
RELEASENAME=${2}
if [ -z "${RELEASENAME}" ]; then
    cmderr "missing name of release to publish"
fi

# Grant access to the various AWS accounts we use for production environments,
# because if the publisher is in a different AWS account than the bucket owner
# then delegating access to other AWS accounts becomes a pain.
grant_pulumi_accts_access() {
    local S3_PATH=${1}
    local BUCKET=$(echo ${S3_PATH} | cut -d/ -f3-3)
    local KEY=$(echo ${S3_PATH} | cut -d/ -f4-)

    # Account identifiers.

    aws s3api put-object-acl \
        --bucket ${BUCKET} \
        --key ${KEY} \
        --grant-full-control ${PULUMI_ACCOUNT_ID_PROD},${PULUMI_ACCOUNT_ID_STAG},${PULUMI_ACCOUNT_ID_TEST},${PULUMI_ACCOUNT_ID_DEV}
}

# Upload the release(s) to our S3 bucket.
PUBPREFIX=s3://eng.pulumi.com/releases/${RELEASENAME}
for target in ${@:3}; do
    PUBTARGET=${PUBPREFIX}/${target}.tgz
    echo Publishing ${RELEASENAME}@${target} to: ${PUBTARGET}
    if [ -z "${FIRSTTARGET}" ]; then
        # Upload the first one for real.
        aws s3 cp ${PUBFILE} ${PUBTARGET}
        grant_pulumi_accts_access ${PUBTARGET}
        FIRSTTARGET=${PUBTARGET}
    else
        # For all others, reuse the first target to avoid re-uploading.
        aws s3 cp ${FIRSTTARGET} ${PUBTARGET}
        grant_pulumi_accts_access ${PUBTARGET}
    fi
done

