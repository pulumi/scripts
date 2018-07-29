#!/bin/bash

# usage compose-publish-nodejs-package <path-to-package-root> <s3-root>
function compose-publish-nodejs-package() {
    local package_root=$1
    local s3_root=$2
    local package_name=$(jq -r '.name' < "${package_root}/package.json" | tr -d "@" | tr "/" "-")
    local temp_file=$(mktemp)
    yarn --cwd "${package_root}" pack --filename "${temp_file}"
    aws s3 cp --acl public-read "${temp_file}" "${s3_root}/nodejs/${package_name}.tgz"
}

# usage compose-publish-nodejs-package <path-to-binary> <s3-root>
function compose-publish-binary() {
    local bin_path=$1
    local s3_root=$2
    echo "aws s3 cp --acl public-read \"${bin_path}\" \"${s3_root}/bin/$(basename ${bin_path})\""
    aws s3 cp --acl public-read "${bin_path}" "${s3_root}/bin/$(basename ${bin_path})"
}
