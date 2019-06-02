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

# special mode for removing maintenance
if [ "$2" == "remove_maintenance" ]; then
  echo -n "deleting $maintIdx.. " && deleteIdx $maintIdx 3 && echo done.
  exit 0
fi

# make sure we're not executed during maintenance, and targetIdx isn't already pointed to alias
indexExist $maintIdx && errorQuit "$maintIdx exist, refuse to run during maintenance."
isCurrent=$(curl -fs -X GET $esUrl/_alias/$alias | jq '."'"$targetIdx"'".aliases.'"$alias"'.is_write_index')
[ $isCurrent ] && echo nothing to do, $targetIdx is already the designated write index for alias $alias. && exit 1

# special mode for restoring siem_alarms from backup
if [ "$2" == "restore" ]; then
  # reindex to backupIdx, this will need to be removed manually
  echo start reindexing documents from $backupIdx to $srcIdx .. this may take a while.
  if ! res=$(reIdx $backupIdx $srcIdx) || failed $res; then 
    echo "cannot reindex $backupIdx to $srcIdx. result:" && echo -e "$res" && exit 1
  fi
  echo done. result: && echo $res
  exit 0
fi

# notify user what will happen
echo "This script will replace $baseIdx index with an alias that points to $targetIdx index.

The steps are:
1 - create $backupIdx and $maintIdx indices, then copy documents from $srcIdx index to them
2 - delete $srcIdx index, then create $alias alias for index pattern $baseIdx-*, 
    and have the write index points to $maintIdx
3 - create $targetIdx index, and copy documents from $maintIdx to it
4 - add $alias alias for index pattern $baseIdx-* to $targetIdx and have 
    the write index points to it
5 - delete $maintIdx

Any update to $srcIdx index that occur during the time between completion of step 1 and step 2" \
"will be lost, so it is recommended that you stop all writes to $srcIdx now before continuing.
"

if ! ask "Are you sure you want to continue?" N; then exit 1; fi

# first check if we can get a matching docs
echo "
** STEP 1:"
echo -n "counting available documents in $srcIdx.. "
count=$(countDoc $srcIdx) || errorQuit "error in counting: $count"
[ "$count" == "0" ] && errorQuit "there is no document found in $srcIdx"
echo "found $count documents."

# create/update backup index
echo -n "creating $backupIdx index.. "
indexExist $backupIdx && out=$(updateIdx $backupIdx) || out=$(createIdx $maintIdx) 
[ ! -z "$out" ] && echo "$out" || echo done.

# reindex to backupIdx, this will need to be removed manually
echo -n "start reindexing documents from $srcIdx to $backupIdx .. this may take a while. "
if ! res=$(reIdx $srcIdx $backupIdx) || failed $res; then 
  echo "cannot reindex $srcIdx to $backupIdx. result:" && echo -e "$res" && exit 1
fi
echo done.

# create/update maintIdx
echo -n "creating $maintIdx index.. "
indexExist $maintIdx && out=$(updateIdx $maintIdx) || out=$(createIdx $maintIdx) 
[ ! -z "$out" ] && echo "$out" || echo done.

# reindex to maintIdx, deleting it upon failure
echo -n "start reindexing documents from $srcIdx to $maintIdx .. this may take a while. "
if ! res=$(reIdx $srcIdx $maintIdx) || failed $res; then 
  echo "cannot reindex $srcIdx to $maintIdx. result:" && echo -e "$res"
  echo -n deleting $maintIdx.. && deleteIdx $maintIdx 5 && echo done. && exit 1
fi
echo done.

# make sure the count is correct
sleep $refreshWaitSec
for i in $backupIdx $maintIdx; do
  echo -n "counting available documents in $i.. "
  c=$(countDoc $i) || errorQuit "error in counting: $c"
  [ "$c" == "0" ] && errorQuit "there is no document found in $i"
  echo "found $c documents."
  (( c < count )) && echo "$c is less than $count docs found in $srcIdx. Deleting it" && \
    deleteIdx $i 5 && echo done. && exit 1
done

echo "** STEP 2:"
# deleting srcIdx
echo -n "deleting $srcIdx prior to adding $alias alias.. "
deleteIdx $srcIdx && echo done

# update alias
echo -n "adding alias $alias for $maintIdx and pointing the write index to it.. "
if ! alres="$(createAlias $maintIdx $alias true)"; then echo "error: $alres" && exit 1; fi
echo done.

echo "** STEP 3:"
# create/update targetIdx
echo -n "creating $targetIdx index.. "
indexExist $targetIdx && out=$(updateIdx $targetIdx) || out=$(createIdx $targetIdx) 
[ ! -z "$out" ] && echo "$out" || echo "done."

# reindex to targetIdx, deleting it upon failure
echo -n "start reindexing documents from $maintIdx to $targetIdx .. this may take a while. "
if ! res=$(reIdx $maintIdx $targetIdx) || failed "$res" ; then 
  echo "cannot reindex $maintIdx to $targetIdx. result:" && echo -e "$res"
  echo -n deleting $targetIdx.. && deleteIdx $targetIdx 5 && echo done. && exit 1
fi
echo done.

# count the docs in targetIdx
echo -n "counting available documents in $targetIdx.. "
c=$(countDoc $i) || errorQuit "error in counting: $c"
[ "$c" == "0" ] && errorQuit "there is no document found in $targetIdx"
echo "found $c documents."
(( c < count )) && echo "$c is less than $count docs found in $srcIdx. Deleting it" && \
  deleteIdx $targetIdx 5 && echo done. && exit 1

echo "** STEP 4:"
# replace alias
echo -n "point alias $alias from $maintIdx to $targetIdx.. "
if ! alres="$(replaceAlias $maintIdx $alias $targetIdx $alias true)"; then echo -e "error: $alres" && exit 1; fi
echo done.

echo "** STEP 5:"
# deleting maintIdx
echo -n "deleting $maintIdx.. " && deleteIdx $maintIdx && echo done.

echo "
Migration to $alias alias completed successfully.
Please verify its content then manually remove siem_alarms_backup if everything is OK.
"
