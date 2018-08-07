#!/bin/bash
export GOOS=linux
mkdir dist
go build -o azcontroller
mv azcontroller dist

echo "Build finished."