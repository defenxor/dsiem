#!/bin/bash
esUrl=$1
cmd=$2
baseIdx="siem_alarms"
alias="siem_alarms"
idxPattern="$baseIdx-*"
srcIdx="$baseIdx"
targetIdx="$baseIdx-current"
maintIdx="${baseIdx}_maintenance"
backupIdx="${baseIdx}_backup"
replica=1
shards=2
refreshWaitSec=3

# tools requirement
for t in curl jq dirname; do 
  command -v $t >/dev/null 2>&1 || { echo the required $t command is not available && exit 1 ;}
done

# move to this script directory
dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"
cd $dir

# extra functions
source ./idx_lib.sh || { echo cannot load es-lib.sh && exit 1; }

[ "$esUrl" == "" ] && echo "Require ES URL as 1st argument." \
"Example: $0 http://elasticsearch:9200" && exit 1

# check ES URL is accessible
esAccessible || { echo "Cannot access ES at $esUrl." \
"Make sure to include the protocol as well as the port, e.g. http://elasticsearch:9200" && exit 1 ;}

# make sure we're not executed during maintenance, and targetIdx is really the correct source
indexExist $maintIdx && errorQuit "$maintIdx exist, refuse to run during maintenance."
isCurrent=$(curl -fs -X GET $esUrl/_alias/$alias | jq '."'"$targetIdx"'".aliases.'"$alias"'.is_write_index')
[ ! $isCurrent ] && echo $targetIdx is not the designated write index for alias $alias. && exit 1

# notify user what will happen
echo "This script will perform force_merge on $targetIdx index to reduce its storage footprint.

The steps are:
1 - create $maintIdx index, then copy documents from $targetIdx index to it
2 - add $alias alias to $maintIdx, and switch the alias write index to it
3 - force_merge $targetIdx to free storage space
4 - copy (reindex) any changes made on $maintIdx to $targetIdx
5 - switch $alias write index back to $targetIdx
6 - copy (reindex) any changes made on $maintIdx to $targetIdx
7 - delete $maintIdx

The reindex processes in 1, 4, and 6 do not reset document version." \
"$targetIdx will have the latest version of all documents by the end of step 6."

if ! ask "Are you sure you want to continue?" N; then exit 1; fi

# first check if we can get a matching docs
echo "
** STEP 1:"
echo -n "counting available documents in $targetIdx.. "
count=$(countDoc $targetIdx) || errorQuit "error in counting: $count"
[ "$count" == "0" ] && errorQuit "there is no document found in $targetIdx"
echo "found $count documents."

# create/update maintIdx
echo -n "creating $maintIdx index.. "
indexExist $maintIdx && out=$(updateIdx $maintIdx) || out=$(createIdx $maintIdx) 
[ ! -z "$out" ] && echo "$out" || echo done.

# reindex to maintIdx, deleting it upon failure
echo -n "start reindexing documents from $targetIdx to $maintIdx .. this may take a while. "
if ! res=$(reIdx $targetIdx $maintIdx) || failed $res; then 
  echo "cannot reindex $targetIdx to $maintIdx. result:" && echo -e "$res"
  echo -n deleting $maintIdx.. && deleteIdx $maintIdx 5 && echo done. && exit 1
fi
echo done.

# make sure the count is correct
sleep $refreshWaitSec
for i in $maintIdx; do
  echo -n "counting available documents in $i.. "
  c=$(countDoc $i) || errorQuit "error in counting: $c"
  [ "$c" == "0" ] && errorQuit "there is no document found in $i"
  echo "found $c documents."
  (( c < count )) && echo "$c is less than $count docs found in $targetIdx. Deleting it" && \
    deleteIdx $i 5 && echo done. && exit 1
done

echo "** STEP 2:"

# update alias
echo -n "point alias $alias from $targetIdx to $maintIdx.. "
if ! alres="$(replaceAlias $targetIdx $alias $maintIdx $alias true)"; then echo "error: $alres" && exit 1; fi
echo done.

echo "** STEP 3:"
# force_merge targetIdx
echo -n "running force_merge on $targetIdx .. this may take a while. "
echo done.

echo "** STEP 4:"
# reindex changes on maintIdx to targetIdx
while true; do
  echo -n "start reindexing documents from $maintIdx to $targetIdx .. this may take a while. "
  if ! res=$(reIdx $maintIdx $targetIdx) || failed "$res" ; then 
    echo "cannot reindex $maintIdx to $targetIdx. result:" && echo -e "$res"
    echo "WARNING: $alias may currently be pointing to $maintIdx."
    echo "Please fix the cause of the above error then retry the operation."
    while ! ask "Are you ready to retry?" Y; do true; done
  else 
    break
  fi
done
echo done.

echo "** STEP 5:"
# replace alias
while true; do
  echo -n "point alias $alias from $maintIdx to $targetIdx.. "
  if ! alres="$(replaceAlias $maintIdx $alias $targetIdx $alias true)"; then 
    echo -e "error: $alres"
    echo "WARNING: $alias may currently be pointing to $maintIdx."
    echo "Please fix the cause of the above error then retry the operation."
    while ! ask "Are you ready to retry?" Y; do true; done
  else 
    break
  fi
done
echo done.

echo "** STEP 6:"
# reindex changes on maintIdx to targetIdx
while true; do
  echo -n "start reindexing documents from $maintIdx to $targetIdx again .. this may take a while. "
  if ! res=$(reIdx $maintIdx $targetIdx) || failed "$res" ; then 
    echo "cannot reindex $maintIdx to $targetIdx. result:" && echo -e "$res"
    echo "Please fix the cause of the above error then retry the operation."
    echo "Alternatively, you can press N to skip this second reindexing."
    if ! ask "Do you want to retry?" Y; then break; fi
  else 
    break
  fi
done
echo done.

echo "** STEP 7:"
# deleting maintIdx
echo -n "deleting $maintIdx.. " && deleteIdx $maintIdx && echo done.

echo "
Force merging $targetIdx completed successfully.
"
