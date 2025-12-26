#!/bin/bash
vsce package
vsce publish

# 进行不兼容的API更改时递增主版本号（第1位）
# vsce publish major
# 进行向后兼容的功能性更改时递增次版本号（第2位）
# vsce publish minor
# 进行向后兼容的问题修正时递增修订号（第3位）
# vsce publish patch