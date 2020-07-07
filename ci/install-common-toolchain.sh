# Travis sources this script, so we can export variables into the
# outer shell, so we don't want to set options like nounset because
# they would be set in the outer shell as well, so do as much logic as
# we can in a subshell.
#
# We do need "OS" to be set in the parent shell, though, since we
# inspect it at the end of the script in order to determine whether or
# not to munge the PATH to include Python executables.
export OS=""
case $(uname) in
    "Linux") OS="linux";;
    "Darwin") OS="darwin";;
    *) echo "error: unknown host os $(uname)" ; exit 1;;
esac

# We also install node outside of the subshell, as nvm needs to export
# variables to the parent shell.
NODE_VERSION="${NODE_VERSION:-12.18.2}"
echo "installing node ${NODE_VERSION}"
nvm install "${NODE_VERSION}"

# We're going to use pip3 to install some tools, and the location
# they are installed to is not on the $PATH by default on OSX Travis
if [ "${OS}" = "darwin" ]; then
    export PATH=$HOME/Library/Python/3.7/bin:$PATH
fi

(
    set -o nounset -o errexit -o pipefail
    [ -e "$(go env GOPATH)/bin" ] || mkdir -p "$(go env GOPATH)/bin"

    YARN_VERSION="${YARN_VERSION:-1.13.0}"
    GOLANGCI_LINT_VERSION="${GOLANGCI_LINT_VERSION:-1.27.0}"
    PIP_VERSION="${PIP_VERSION:-10.0.0}"
    VIRTUALENV_VERSION="${VIRTUALENV_VERSION:-15.2.0}"
    PIPENV_VERSION="${PIPENV_VERSION:-2018.11.26}"
    AWSCLI_VERSION="${AWSCLI_VERSION:-1.16.304}"
    WHEEL_VERSION="${WHEEL_VERSION:-0.33.6}"
    TWINE_VERSION="${TWINE_VERSION:-1.13.0}"
    TF2PULUMI_VERSION="${TF2PULUMI_VERSION:-0.7.0}"
    PANDOC_VERSION="${PANDOC_VERSION:-2.6}"
    PULUMICTL_VERSION="${PULUMICTL_VERSION:-0.0.5}"

    # jq isn't present on OSX, but we use it in some of our scripts. Install it.
    if [ "${OS}" = "darwin" ]; then
        brew update
        brew install jq
    fi

    # On Linux Travis, System Python is 2.7.X, use `pyenv` to pick up Python 3.6.7
    if [ "${OS}" = "linux" ]; then
        pyenv versions
        pyenv global 3.6.7
    fi    

    echo "installing yarn ${YARN_VERSION}"
    curl -o- -L https://yarnpkg.com/install.sh | bash -s -- --version "${YARN_VERSION}"

    echo "installing GolangCI-Lint ${GOLANGCI_LINT_VERSION}"
    curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b "$(go env GOPATH)/bin" "v${GOLANGCI_LINT_VERSION}"

    echo "installing gocovmerge"

    # gocovmerge does not publish versioned releases, but it also hasn't been updated in two years, so
    # getting HEAD is pretty safe.
    GO111MODULE=off go get -v github.com/wadey/gocovmerge

    echo "installing pipenv ${PIPENV_VERSION}"
    pip3 install --user "pipenv==${PIPENV_VERSION}"

    echo "installing AWS cli ${AWSCLI_VERSION}"
    pip3 install --user "awscli==${AWSCLI_VERSION}"

    echo "installing Wheel and Twine, so we can publish Python packages"
    pip3 install --user "wheel==${WHEEL_VERSION}" "twine==${TWINE_VERSION}"

    echo "installing pandoc, so we can generate README.rst for Python packages"
    if [ "${OS}" = "linux" ]; then
        curl -sfL -o /tmp/pandoc.deb "https://github.com/jgm/pandoc/releases/download/${PANDOC_VERSION}/pandoc-${PANDOC_VERSION}-1-amd64.deb"
        sudo apt-get install /tmp/pandoc.deb
    else
        # This is currently version 2.6 - we'll likely want to track the version
        # in brew pretty closely in CI, as it's a pain to install otherwise.
        brew install pandoc
    fi

    echo "installing dotnet sdk and runtime"
    if [ "${OS}" = "linux" ]; then
        wget -q https://packages.microsoft.com/config/ubuntu/18.04/packages-microsoft-prod.deb -O packages-microsoft-prod.deb
        sudo dpkg -i packages-microsoft-prod.deb
        sudo add-apt-repository universe
        sudo apt-get update
        sudo apt-get install apt-transport-https
        sudo apt-get update
        sudo apt-get install dotnet-sdk-3.1
        sudo apt-get install aspnetcore-runtime-3.1
    else
        brew cask install dotnet-sdk
    fi

    echo "installing Terraform-to-Pulumi conversion tool (${TF2PULUMI_VERSION}-${OS})"
    curl -L "https://github.com/pulumi/tf2pulumi/releases/download/v${TF2PULUMI_VERSION}/tf2pulumi-v${TF2PULUMI_VERSION}-${OS}-x64.tar.gz" | \
			tar -xvz -C "$(go env GOPATH)/bin"
			
    echo "installing Pulumictl utility tool (${PULUMICTL_VERSION}-${OS})"
    curl -L "https://github.com/pulumi/pulumictl/releases/download/v${PULUMICTL_VERSION}/pulumictl-v${PULUMICTL_VERSION}-${OS}-amd64.tar.gz" | \
			tar -xvz -C "$(go env GOPATH)/bin"

    echo "installing gomod-doccopy"
    GO111MODULE=off go get -v github.com/pulumi/scripts/gomod-doccopy
)

# If the sub shell failed, bail out now.
[ "$?" -eq 0 ] || exit 1

# By default some tools are not on the PATH, let's fix that

# Add yarn to the $PATH
export PATH=$HOME/.yarn/bin:$PATH

