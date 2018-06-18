#!/usr/bin/false -- must be sourced

export PULUMI_FAILED_TESTS_DIR=$(mktemp -d)
echo "keeping failed tests at: ${PULUMI_FAILED_TESTS_DIR}"

# after tests, call ./upload-failed-tests
