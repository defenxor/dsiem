#!/bin/bash

mkdir -p ./test/pprof_results

dirs=$(ssh mgmt184 -C "sudo kubectl exec dsiem-0 -c dsiem -- /bin/bash -c 'ls -d /tmp/profile*'" 2>/dev/null)
for d in ${dirs}; do
  echo processing $d ..
  ssh mgmt184 -C "sudo kubectl cp -c dsiem dsiem-0:${d} ${d}" >/dev/null 2>&1
  scp -r mgmt184:$d ./test/pprof_results/
  n=$(basename ${d})
  go tool pprof --pdf ./dsiem ./test/pprof_results/${n}/*.pprof > ./test/pprof_results/${n}/${n}.pdf
done
