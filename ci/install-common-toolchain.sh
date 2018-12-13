nvm install ${NODE_VERSION-v8.11.1}

# Travis sources this script, so we can export variables into the
# outer shell, so we don't want to set options like nounset because
# they would be set in the outer shell as well, so do as much logic as
# we can in a subshell.
(
    set -o nounset -o errexit -o pipefail
    [ -e "$(go env GOPATH)/bin" ] || mkdir -p "$(go env GOPATH)/bin"

    YARN_VERSION="1.3.2"
    DEP_VERSION="0.5.0"
    GOMETALINTER_VERSION="2.0.3"
    GOLANGCI_LINT_VERSION="1.12"
    PIP_VERSION="10.0.0"
    VIRTUALENV_VERSION="15.2.0"
    PIPENV_VERSION="2018.10.13"
    AWSCLI_VERSION="1.14.30"
    WHEEL_VERSION="0.30.0"
    TWINE_VERSION="1.9.1"

    OS=""
    case $(uname) in
        "Linux") OS="linux";;
        "Darwin") OS="darwin";;
        *) echo "error: unknown host os $(uname)" ; exit 1;;
    esac

    # jq isn't present on OSX, but we use it in some of our scripts. Install it.
    if [ "${TRAVIS_OS_NAME:-}" = "osx" ]; then
        brew install jq
    fi

    echo "installing yarn ${YARN_VERSION}"
    curl -o- -L https://yarnpkg.com/install.sh | bash -s -- --version ${YARN_VERSION}

    echo "installing dep ${DEP_VERSION}"
    curl -L -o "$(go env GOPATH)/bin/dep" https://github.com/golang/dep/releases/download/v${DEP_VERSION}/dep-${OS}-amd64
    chmod +x "$(go env GOPATH)/bin/dep"

    echo "installing Gometalinter ${GOMETALINTER_VERSION}"
    curl -L "https://github.com/alecthomas/gometalinter/releases/download/v${GOMETALINTER_VERSION}/gometalinter-v${GOMETALINTER_VERSION}-${OS}-amd64.tar.bz2" | tar -jxv --strip-components=1 -C "$(go env GOPATH)/bin"

    chmod +x "$(go env GOPATH)/bin/gometalinter"
    chmod +x "$(go env GOPATH)/bin/linters/"*

    # Gometalinter looks for linters on the $PATH, so let's move them out
    # of the linters folder and into GOBIN (which we know is on the $PATH)
    mv "$(go env GOPATH)/bin/linters/"* "$(go env GOPATH)/bin/."
    rm -rf "$(go env GOPATH)/bin/linters/"

    echo "installing GolangCI-Lint ${GOLANGCI_LINT_VERSION}"
    curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b "$(go env GOPATH)/bin" "v${GOLANGCI_LINT_VERSION}"

    echo "installing gocovmerge"

    # gocovmerge does not publish versioned releases, but it also hasn't been updated in two years, so
    # getting HEAD is pretty safe.
    go get -v github.com/wadey/gocovmerge

    echo "upgrading Pip to ${PIP_VERSION}"
    sudo pip install --upgrade "pip>=${PIP_VERSION}"
    pip install --user --upgrade "pip>=${PIP_VERSION}"

    echo "installing virtualenv ${VIRTUALENV_VERSION}"
    sudo pip install "virtualenv==${VIRTUALENV_VERSION}"
    pip install --user "virtualenv==${VIRTUALENV_VERSION}"

    echo "installing pipenv ${PIPENV_VERSION}"
    pip install --user "pipenv==${PIPENV_VERSION}"

    echo "installing AWS cli ${AWSCLI_VERSION}"
    pip install --user "awscli==${AWSCLI_VERSION}"

    echo "installing Wheel and Twine, so we can publish Python packages"
    pip install --user "wheel==${WHEEL_VERSION}" "twine==${TWINE_VERSION}"

    echo "installing pandoc, so we can generate README.rst for Python packages"
    if [ "${TRAVIS_OS_NAME:-}" = "linux" ]; then
        # We've seen cases in Travuis where `apt-get update` fails because
        # some sources couldn't be refreshed. Instead of failing the entire
        # operation eagerly, allow this command to fail. If we don't refresh
        # enough sources to install software later, we'll blow up then.
        sudo apt-get update || true
        sudo apt-get install pandoc
    else
        brew install pandoc
    fi
)

# If the sub shell failed, bail out now.
[ "$?" -eq 0 ] || exit 1

# By default some tools are not on the PATH, let's fix that

# On OSX, the location that pip installs helper scripts to isn't on the path
if [ "${TRAVIS_OS_NAME:-}" = "osx" ]; then
    export PATH=$PATH:$HOME/Library/Python/2.7/bin
fi

# Add yarn to the $PATH
export PATH=$HOME/.yarn/bin:$PATH
