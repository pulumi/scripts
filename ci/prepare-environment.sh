# Travis sources this script, so we can export variables into the
# outer shell, so we don't want to set options like nounset because
# they would be set in the outer shell as well, so do as much logic as
# we can in a subshell.

export PULUMI_SCRIPTS="$(go env GOPATH)/src/github.com/pulumi/scripts"

(
    set -o nounset -o errexit -o pipefail

    if [ "${TRAVIS_OS_NAME:-}" = "osx" ]; then
        sudo mkdir /opt/pulumi
        sudo chown "${USER}" /opt/pulumi
    fi

    # If we have an NPM token, put it in the .npmrc file, so we can use it:
    if [ ! -z "${NPM_TOKEN:-}" ]; then
        echo "//registry.npmjs.org/:_authToken=\${NPM_TOKEN}" > ~/.npmrc
    fi

    # Put static entries for Pulumi backends in /etc/hosts
    "${PULUMI_SCRIPTS}/ci/pulumi-hosts" | sudo tee -a /etc/hosts

    # Multiple copies of `pipenv` can race to create `~/.local/share/virtualenvs` and when they do so one will fail and trigger a crash inside pipenv itself. To work around this issue, create this directory upfront.
    mkdir -p "${HOME}/.local/share/virtualenvs"
) || exit 1  # Abort outer script if subshell fails.

export PULUMI_ROOT=/opt/pulumi
