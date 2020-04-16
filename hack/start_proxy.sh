#!/bin/bash

server="./build/proxy"
let item=0
item=`ps -ef | grep $server | grep -v grep | wc -l`

if [ $item -eq 1 ]; then
        echo "The p2p-proxy is running, shut it down..."
        pid=`ps -ef | grep $server | grep -v grep | awk '{print $2}'`
        kill -9 $pid
fi

echo "Start p2p-proxy now ..."
make src.build
./build/proxy  -rootdir /data/bt/proxy -seeders x.x.x.x:65005 -trackers http://x.x.x.x:6969/announce -rule x.x.x.x -verbose >> ./p2p-proxy.log 2>&1 &