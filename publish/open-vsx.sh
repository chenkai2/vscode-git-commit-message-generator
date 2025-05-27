#!/bin/bash
workspaceFolder=$(cd $(dirname $0);cd ..;pwd)
cd $workspaceFolder
name=$(grep '"name":' package.json | head -1 | awk -F: '{print $2}' | sed 's/[", ]//g')
version=$(grep '"version":' package.json | head -1 | awk -F: '{print $2}' | sed 's/[", ]//g')
packageName="${name}-${version}.vsix"
echo $packageName
ovsx publish $packageName
