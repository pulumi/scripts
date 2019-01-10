# Pulumi Scripts

This repository stores all ancillary scripts for Pulumi's various repositories.

## Updating

All of our repositories check out the tip of master to get the scripts versions. Sometimes its helpful to "flight" changes to the scripts when developing new changes. The easiest way we've found to do this is the following workflow:

1. Push a topic branch to this repository (e.g. `ellismg/update-yarn-version`)
2. In an existing repo, edit the `.travis.yml` file to include a call to checkout this topic branch. You can do this by adding another step in the `before_install` section, right after the call to git clone to clone this repository. For example `git -C "${GOPATH}/src/github.com/pulumi/scripts" checkout ellismg/update-yarn-version`. Note the use of `-C` here to ensure `git` runs the checkout in the scripts repository instead of the current repository.
3. Open a PR and let CI run.

Once the changes are working, you can merge the script changes into master and things will be picked up on future runs. You'll then be able to abandon the PR opened in step (2) above.



