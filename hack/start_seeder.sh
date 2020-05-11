#!/bin/bash

server="./build/seeder"
let item=0
item=`ps -ef | grep $server | grep -v grep | wc -l`

if [ $item -eq 1 ]; then
	echo "The p2p-seeder is running, shut it down..."
	pid=`ps -ef | grep $server | grep -v grep | awk '{print $2}'`
	kill -9 $pid
fi

echo "Start p2p-seeder now ..."
make build
./build/seeder -config examples/seeder/config.yaml >> ./p2p-seeder.log 2>&1 &