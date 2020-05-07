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
./build/proxy -config examples/proxy/config.yaml >> ./p2p-proxy.log 2>&1 &