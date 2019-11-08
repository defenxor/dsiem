#!/bin/bash

usage () {
  echo "
Requires 3 parameters <ip> <port> <path_to_csv_file>
"
}

[ -z "$1" ] || [ -z "$2" ] || [ -z "$3" ] && { usage; exit 1; }

ip=$1
port=$2
file=$3

touch $file || exit $?

# header first
head -1 $file | grep -q "Plugin ID" || echo '"Plugin ID","CVE","CVSS","Risk","Host","Protocol","Port","Name","Synopsis","Description","Solution","See Also","Plugin Output"' > $file || { echo failed to write header; exit 1; }

echo '77823,"CVE-2014-6271",9.8,"High","'"$ip"'","http",'"$port"',"Bash Remote Code Execution (Shellshock)","A system shell on the remote host is vulnerable to command injection.","The remote host is running a version of Bash that is vulnerable to command injection via environment variable manipulation. Depending on the configuration of the system, an attacker could remotely execute arbitrary code","Update Bash","http://www.nessus.org/u?dacf7829",' >> $file || { echo failed to write data for $ip:$port; exit 1; }

