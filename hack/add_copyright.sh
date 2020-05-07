#!/bin/bash

for i in $(find ./ -name '*.go')
do
  if ! grep -q Copyright $i
  then
    cat hack/copyright.txt $i >$i.new && mv $i.new $i
  fi
done
