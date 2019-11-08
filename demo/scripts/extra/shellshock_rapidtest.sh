#!/bin/bash

[ -z "$1" ] && echo need second as first argument

first=5
[ ! -z "$2" ] && first=$2
./exploit.sh localhost 30001 /dev/null
for i in {1..4}; do
  ./exploit.sh localhost 30001 /var/www/html/index${i}.html
 [ "$i" == 1 ] && sleep $first || sleep $1
done
