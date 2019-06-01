#!/bin/bash

# this script is just to help new devs to get familiar with the workflow.

cmd=$1
param=$2

if [ -z "$cmd" ] || [ "$cmd" == "help" ]; then cat <<EOF
Use the following as 1st argument:
  start        : creates a new temporary branch in local and remote repo. You will then use this branch during
                 development before submitting pull request, and to push improvements during the PR review process.
  commit       : commit your changes to the local branch.
  push         : push your committed changes to the remote branch. This shall trigger a CI build-test job unless 
                 a "[skip ci]" text is added to one of the commit messages.
  sync         : merge changes from master (if any) to your working branch, so you dont develop on top of an outdated
                 copy of master.
  ci-status    : view github status checks of the latest commit you push on the branch.
  pull-request : create a pull-request on github, which will start the build-test CI job and review process.
                 you can add more commits and pushes to update the PR based on review feedbacks or CI test results.
  cleanup      : after the PR has been merged (either by maintainers or bot), you can use this to delete the local 
                 working and remote branch.
  
EOF
  exit 1
fi

command -v hub >/dev/null 2>&1 || (echo cannot find hub command, please install it for your OS from https://github.com/github/hub/releases && exit 1)

if ! [[ "$cmd" =~ ^(start|sync|commit|push|pull-request|cleanup|ci-status)$ ]]; then 
  echo need start, sync, commit, push, pull-request, or cleanup, as first argument && exit 1
fi

# start creates your working branch that will be the source of pull request later on
if [ "$cmd" == "start" ]; then
  [ -z "$param" ] && echo need a branch name as 2nd argument. && exit 1
  hub sync && \
  git checkout -b $param master && \
  git push -u origin $param
  exit $?
fi

# the rest of the commands shouldn't be used on master branch
thisbranch=$(git rev-parse --abbrev-ref HEAD)
[ "$thisbranch" == "master" ] && echo cannot use $1 on master branch, please create a working branch first && exit 1

# sync will sync this branch will changes in origin/master, so you dont develop on out-of-date copies
if [ "$cmd" == "sync" ]; then
  # sync others commit to this branch first
  git pull --rebase && \
  # sync master
  git checkout master && \
  git pull --rebase && \
  # put all of your changes on this branch on top of master
  git checkout $thisbranch && \
  git rebase master && \
  echo note you may need to use git force --push to sync your local branch to the remote
  exit $?
fi

# cleanup delete the remote and local branch when there's no open pr for it
if [ "$cmd" == "cleanup" ]; then
  open=$(hub pr list -b $thisbranch)
  [ "$open" != "" ] && echo cannot continue due to open pr: "$open" && exit 1
  git checkout master
  git branch -D $thisbranch && \
  git push origin --delete $thisbranch 2>/dev/null || true
  exit $?
fi

if [ "$cmd" == "ci-status" ]; then
  hub ci-status -v
  exit $?
fi

# for the rest of useful commands, just pass it directly to hub
if [ "$cmd" == "pull-request" ] || [ "$cmd" == "push" ] || [ "$cmd" == "commit" ]; then
  hub $cmd
  exit $?
else 
  echo $cmd is not a supported command
  exit 1
fi
