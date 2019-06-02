#!/bin/bash
esUrl=$1
dt=$2
minDays=2
baseIdx="siem_alarms"
alias="siem_alarms"
srcIdx="$baseIdx-current"
maintIdx="${baseIdx}_maintenance"
replica=0
shards=2

datecheck() {
    local format="$1" d="$2"
    [[ "$(date "+$format" -d "$d" 2>/dev/null)" == "$d" ]]
}

datediff() {
    d1=$(date -d "$1" +%s)
    d2=$(date -d "$2" +%s)
    echo $(( (d1 - d2) / 3600 )) # in hours
}

# sanity checks

for t in curl jq date; do 
  command -v $t >/dev/null 2>&1 || { echo the required $t command is not available && exit 1 ;}
done

exampleDt=$(date +"%Y-%m-%d" -d "$minDays days ago")
[ "$dt" == "" ] || [ "$esUrl" == "" ] && \
echo 'require ES URL as 1st argument, and date (yyyy-mm-dd) as 2nd argument' && \
echo example: $0 http://elasticsearch:9200 $exampleDt && exit 1

datecheck "%Y-%m-%d" "$dt"
[ "$?" != "0" ] && echo $dt is not a valid date. Example valid one: $exampleDt etc. && exit 1

now=$(date +"%Y-%m-%d")
hoursDiff=$(datediff $now "$dt")
daysDiff=$(( hoursDiff / 24 ))
if [[ $daysDiff -lt $minDays ]]; then
  echo "this tool should only be executed against documents that are at least $minDays days old"
  echo "it means the earliest date you can supply is $exampleDt."
  exit 1
fi

# check ES URL is accessible
out=$(curl -fsS -X GET "$esUrl" 2>&1 | grep -q "You Know, for Search")
[ "$?" != "0" ] && echo "cannot access ES at $esUrl." && \
echo "Make sure to include the protocol as well as the port, e.g. http://elasticsearch:9200" && exit 1

# make sure we're not executed during maintenance, i.e. srcIdx is really the correct source for reindex
curl -fs -I "$esUrl/$maintIdx" -o /dev/null 2>&1
[ "$?" == "0" ] && echo $maintIdx exist, refused to run during maintenance. && exit 1
isCurrent=$(curl -fs -X GET $esUrl/_alias/$alias | jq '."'"$srcIdx"'".aliases.'"$alias"'.is_write_index')
[ ! $isCurrent ] && echo $srcIdx is not the designated write index for alias $alias. && exit 1

# notify user what will happen
postfix=$(echo $dt | sed 's/-/\./g')
targetIdx=${baseIdx}-${postfix}
timegte="${dt}T00:00:00Z"
daybefore=$(date --date="$dt +1 day" +%Y-%m-%d)
timelt="${daybefore}T00:00:00Z"

echo "This script will move documents from $srcIdx to $targetIdx.

The steps are:
1 - query $srcIdx for documents whose timestamp is ≥ $timegte and < $timelt
2 - create index $targetIdx if it doesnt exist yet
3 - reindex documents found in step 1 to $targetIdx
4 - delete those documents from $srcIdx

Execution will automatically continue in 10 seconds. Press CTRL-C now to abort."
sleep 10

echo "
** STEP 1:"
# first check if we can get a matching docs
echo -n "counting target documents .. "
count=$(curl -fsS -X GET "$esUrl/$srcIdx/_count" -H 'Content-Type: application/json' -d'
{ "query": { "range" : { "@timestamp" : { "gte": "'"${timegte}"'", "lt": "'"${timelt}"'" } } } }' | jq ".count")
[ "$?" != "0" ] && echo cannot count the number of documents updated on ${dt}. && exit 1

[ "$count" == "0" ] && echo there is no matching document found in $srcIdx. && exit 0
echo found $count documents.

echo "** STEP 2:"
# create/update targetIdx
curl -fs -I "$esUrl/$targetIdx" -o /dev/null
if [ "$?" != "0" ]; then
  # targetIdx doesn't exist, create it
  echo cannot find index $targetIdx, creating it with replica set to $replica.
  out=$(curl -f -X PUT "$esUrl/$targetIdx" -H 'Content-Type: application/json' -d'
  { "settings" : { "index" : { "number_of_shards" : '"$shards"', "number_of_replicas" : '"$replica"' }}}' 2>&1)
  [ "$?" != "0" ] && echo cannot create $targetIdx. Message: $out && exit 1
else
  # targetIdx already exist
  echo found existing index $targetIdx, ensuring its replica set to $replica.
  out=$(curl -f -X PUT "$esUrl/$targetIdx/_settings" -H 'Content-Type: application/json' -d'
  { "index" : { "number_of_replicas" : 0 }}' 2>&1 )
  [ "$?" != "0" ] && echo cannot update $targetIdx replica setting. Message: $out && exit 1
fi

echo "** STEP 3:"
# start reindex
echo start reindexing documents from $srcIdx to $targetIdx .. this may take a while.
gdt="$dt"
out=$(curl -f -sS -X POST "$esUrl/_reindex" -H 'Content-Type: application/json' -d'
{
  "source": {
    "index": "'"$srcIdx"'",
    "query": {
      "range" : {
        "@timestamp" : {
          "gte": "'"$timegte"'",
          "lt": "'"$timelt"'"
        }
      }
    }
  },
  "dest": {
    "index": "'"$targetIdx"'"
  }
}
' | jq ".")

failures=$(echo "$out" | jq ".failures")

# check for failure
if [ "$failures" != '[]' ]; then
  # fail to _reindex, so we delete targetIdx and exit with non-zero status
  echo Failure detected, _reindex API returns the following status: 
  echo "$out"
  echo "will now delete $targetIdx to prevent duplicates."
  while true; do
    out=$(curl -fsS -X DELETE "$esUrl/$targetIdx")
    [ "$?" == "0" ] && break
    # failure to delete may means the idx no longer exist
    out=$(curl -fsS -I "$esUrl/$targetIdx" 2>&1)
    if [ "$?" != "0" ] && echo $out | grep -q 404; then 
       # curl success and there's 404 in output means idx doesn't exist
       break
    fi
    echo "Cannot delete $targetIdx or access $esUrl, retrying in 5s .."
    sleep 5
  done
  echo $targetIdx deleted. Returning non-zero to indicate reindexing failure
  return 2
else
  # _reindex succeed, now delete the old docs in srcIdx
  echo "successfully reindex $count documents."
  echo "reindex result: "
  echo "$out"
  echo "** STEP 4:"
  echo "deleting documents from $srcIdx .."
  out=$(curl -fsS -X POST "$esUrl/$srcIdx/_delete_by_query" -H 'Content-Type: application/json' -d'
  {
    "query": {
      "range" : {
        "@timestamp" : {
          "gte": "'"$timegte"'",
          "lt": "'"$timelt"'"
        }
      }
    }
  }
  ' | jq ".")
  deleted=$(echo "$out" | jq ".deleted")
  failures=$(echo "$out" | jq ".failures")
  if [ "$deleted" == "$count" ] && [ "$failures" == "[]" ]; then 
    echo successfully delete $deleted documents from $srcIdx.
    echo reindex process complete.
    exit 0
  else
    echo can only delete $deleted documents out of the $count target.
    echo deletion result:
    echo "$out"
    echo exiting with non-zero status to indicate failure.
    exit 2
  fi
fi
