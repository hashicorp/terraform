#!/usr/bin/env bash
#
# This script has various functions that help with building and executing the
# equivalence tests after a successful release.
#
# As of 2022/09/15, the build.yml GitHub actions workflow executes on pushes
# to the main branch, any release branches (such as v1.1, v1.2, etc.), and any
# branch that starts with build-workflow-dev/ for testing. In addition to the
# branches, the workflow also executes on the push of any new tag that matches
# "v[0-9]+.[0-9]+.[0-9]+*" (as in, any tag that matches a new release).
#
# For the time being, the new tag is created as part of the release process in
# terraform-releases as the release process is currently ran from that alternate
# repository. This will change with the introduction of the Common Release
# Tooling (CRT) and as such, we want to build the equivalence testing into the
# new release process rather than the old one. Therefore, the equivalence tests
# are executed by the build.yml workflow whenever a new tag that matches the
# criteria is pushed but only when a tag is pushed and not when a branch is
# pushed.
#
# The equivalence tests are used as a way to track changes made in a release
# automatically, and can be compared against the CHANGELOG for a given release
# in order to improve confidence that all changes are expected/reported.

# git_ref_is_tag returns true if the argument passed in $1 matches the git
# reference for a tag (eg. refs/tag/v1.2.3).
#
# This function should be used to decide whether the equivalence tests will
# execute at all (as they only run on new tags and not updates to branches).
function git_ref_is_tag {
  if [[ -z $GIT_REF ]]; then
    echo "missing GIT_REF environment variable"
    exit 1
  fi

  if [[ "$GIT_REF" =~ "refs/tags" ]]; then
    echo true
  else
    echo false
  fi
}

# branch_for_release_tag returns the git branch that should be used for a
# given tag. The tags include major, minor, and patch versions and sometimes
# prerelease metadata. The release branches only contain major and minor
# versions, so we map between tags and branches in this function.
#
# A special shoutout to alpha releases which happen on the main branch and not
# on a special release branch. This means that the equivalence test updates to
# the main branch will not always make sense in isolation and engineers will
# have to do manual diffs between the main branch and the last relevant release
# branch to see meaningful changes for the alpha releases.
function get_target_branch {
  if [[ -z $PRERELEASE || -z $MAJOR_VERSION || -z $MINOR_VERSION ]]; then
    echo "missing one of [ PRERELEASE=$PRERELEASE , MAJOR_VERSION=$MAJOR_VERSION , MINOR_VERSION=$MINOR_VERSION ]"
    exit 1
  fi

  # TODO(liamcervante): Fix this once the CRT has been launched.
  # alpha releases are performed against the main branch, so the diff for the
  # golden files on the main branch will be nonsensical when looked at in
  # isolation.
  if [[ "$PRERELEASE" == *"alpha"* ]]; then
    echo "main"
    return 0
  fi
  echo "v$MAJOR_VERSION.$MINOR_VERSION"
}

# update_goldens is where the magic happens. This function will use the
# pre-built terraform binary provided in the arguments to run all the
# equivalence tests and update the golden files.
#
# It is assumed the correct branch has been checked out prior to executing this
# function, and that this function is being called from the repository root.
function update_goldens {
  if [[ -z $BASE_VERSION || -z $TERRAFORM_BINARY_PATH ]]; then
    echo "missing one of [ BASE_VERSION=$BASE_VERSION , TERRAFORM_BINARY_PATH=$TERRAFORM_BINARY_PATH ]"
    exit 1
  fi

  pushd tools/equivalence_tests || exit

  go run main.go -tests=testing -goldens=goldens -binary="$TERRAFORM_BINARY_PATH" -update

  changed=$(git diff --quiet -- goldens || echo true)
  if [[ $changed == "true" ]]; then
    git add ./goldens
    git commit -m"Updated equivalence test golden files for $BASE_VERSION."
    git push
  fi

  popd || exit
}
