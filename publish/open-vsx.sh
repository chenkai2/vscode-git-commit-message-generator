#!/bin/bash
name=$(grep '"name":' ../package.json | head -1 | awk -F: '{print $2}' | sed 's/[", ]//g')
version=$(grep '"version":' ../package.json | head -1 | awk -F: '{print $2}' | sed 's/[", ]//g')
packageName="${name}-${version}.vsix"
ovsx publish $packageName
