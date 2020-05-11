#!/bin/bash

server="./build/tracker"
let item=0
item=`ps -ef | grep $server | grep -v grep | wc -l`

if [ $item -eq 1 ]; then
        echo "The tracker is running, shut it down..."
        pid=`ps -ef | grep $server | grep -v grep | awk '{print $2}'`
        kill -9 $pid
fi

echo "Start tracker now ..."
/build/tracker --config dist/example_config.yaml --debug >> ./tracker.log 2>&1 &