#!/bin/bash

scriptdir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
cd $scriptdir

function list_include_item () {
  local list="$1"
  local item="$2"
  if [[ $list =~ (^|[[:space:]])"$item"($|[[:space:]]) ]] ; then
    # yes, list include item
    result=0
  else
    result=1
  fi
  return $result
}

for c in vagrant; do
 command -v $c >/dev/null || { echo -e "\ncannot find a required command: $c"; exit 1; }
done

vm_types="alpine ubuntu"

while ! list_include_item "$vm_types" "$vm_type"; do
  read -p "Which type of VM you want to build? [$vm_types]: " vm_type
done

cd $vm_type && \
  echo "** Running vagrant up .." && \
  vagrant up && \
  echo "** Shutting down VM .." && \
  vagrant halt && \
echo "
Done. Now you can use the VM directly through virtualbox GUI.

Alternatively, you can also use vagrant to start the demo:
$ cd $vm_type && vagrant up && vagrant ssh -c 'su - demo'

"