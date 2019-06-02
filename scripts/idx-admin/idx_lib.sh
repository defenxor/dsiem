#!/bin/bash

# esUrl="${esUrl:-localhost:9200}"

errorQuit() {
  echo "$@"; exit 1; 
}
notify() {
  echo "$@"
}

indexExist() {
  curl -fs -I "$esUrl/$1" -o /dev/null
  return $?
}

esAccessible() {
  out=$(curl -fsS -X GET "$esUrl" 2>&1 | grep -q "You Know, for Search")
  return $?
}

# create/update maintIdx
createIdx () {
  # $1: Index to create
  # $2: shard
  # $3: replica
  out=$(curl -sSf -X PUT "$esUrl/$1" -H 'Content-Type: application/json' -d'
  { "settings" : { "index" : { "number_of_shards" : '"$2"', "number_of_replicas" : '"$3"' }}}' 2>&1)
  ret=$?
  [ "$ret" == "0" ] && echo $out
  return $ret
}

updateIdx() {
  # $1: Index to update
  # $2: shard
  # $3: replica
  out=$(curl -f -X PUT "$esUrl/$1/_settings" -H 'Content-Type: application/json' -d'
  { "index" : { "number_of_shards" : '"$2"', "number_of_replicas" : '"$3"' }}' 2>&1 )
  ret=$?
  [ "$ret" == "0" ] && echo $out
  return $ret
}

reIdx() {
  # $1: src
  # $2: dst
  out=$(curl -f -sS -X POST "$esUrl/_reindex" -H 'Content-Type: application/json' -d'
  {
    "conflicts": "proceed",
    "source": {
    "index": "'"$1"'"
    },
    "dest": {
    "index": "'"$2"'",
    "version_type": "external"
    }
  }' 2>&1)
  ret=$?
  echo -e "$out"
  return $ret
}

forceMerge() {
  curl -fsS -X POST "$esUrl/$1/_forcemerge"
}

countDoc() {
  # $1: idx
  out=$(curl -fsS -X GET "$esUrl/$1/_count" -H 'Content-Type: application/json' 2>&1)
  ret=$?
  [ "$ret" == "0" ] && ( count=$(echo "$out" | jq ".count") && echo $count) || echo $out
  return $ret
}


failed() {
  # $1: API output that has failures field
  failures=$(echo "$1" | jq ".failures" 2>&1)
  [ "$failures" == "[]" ] && return 1 || return 0
}

total() {
  # $1: API output that has total field
  ttl=$(echo "$1" | jq ".total" 2>&1)
  echo $ttl
  [ "$ttl" == "" ] && return 1 || return 0
}

deleteIdx() {
  # $1: idx, $2: second to wait
  waitSec="${2:-3}"
  while true; do
    out=$(curl -fsS -X DELETE "$esUrl/$1" 2>&1)
    [ "$?" == "0" ] && break
    # failure to delete may means the idx no longer exist
    out=$(curl -fsS "$esUrl/$1" 2>&1)
    if [ "$?" != "0" ] && echo "$out" | grep -q 404; then 
       # curl success and there's 404 in output means idx doesn't exist
       echo "$1 doesn't exist"
       break
    fi
    echo "fail to delete $1, retrying in ${waitSec}s.."
    sleep $waitSec
  done
}

createAlias() {
  # $1: alias1 index
  # $2: alias1 name
  # $3: alias1 is_write_index
  out=$(curl -fsS -X POST "$esUrl/_aliases" -H 'Content-Type: application/json' -d'
  {
    "actions" : [
        { 
          "add" : { 
            "index" : "'"$1"'", "alias" : "'"$2"'",
            "is_write_index" : '"$3"'
          } 
        }
    ]
  }' 2>&1)
  ret=$?
  echo "$out"
  return $ret
}


replaceAlias() {
  out=$(curl -fsS -X POST "$esUrl/_aliases" -H 'Content-Type: application/json' -d'
  {
    "actions" : [
        { 
          "remove" : { 
            "index" : "'"$1"'", "alias" : "'"$2"'"
          } 
        },
        { 
          "add" : { 
            "index" : "'"$3"'", "alias" : "'"$4"'",
            "is_write_index" : '"$5"'
          } 
        }
    ]
  }' 2>&1)
  ret=$?
  [ "$ret" != "0" ] && echo "$out"
  return $ret
}

ask() {
  # http://djm.me/ask
  while true; do
    if [ "${2:-}" = "Y" ]; then
      prompt="Y/n"
      default=Y
    elif [ "${2:-}" = "N" ]; then
      prompt="y/N"
      default=N
    else
      prompt="y/n"
      default=
    fi
    # Ask the question - use /dev/tty in case stdin is redirected from somewhere else
    read -p "$1 [$prompt] " REPLY </dev/tty
    # Default?
    if [ -z "$REPLY" ]; then
      REPLY=$default
    fi
    # Check if the reply is valid
    case "$REPLY" in
      Y*|y*) return 0 ;;
      N*|n*) return 1 ;;
    esac
  done
}

