#!/usr/bin/env bash

ROOT=$(git rev-parse --show-toplevel)
dep ensure
cp $ROOT/third_party/linuxha/bottlerocket/*.h $ROOT/vendor/github.com/rmrobinson/bottlerocket-go/
cp $ROOT/third_party/linuxha/bottlerocket/*.c $ROOT/vendor/github.com/rmrobinson/bottlerocket-go/
bazel run //:gazelle
buildozer 'remove clinkopts' //vendor/github.com/rmrobinson/bottlerocket-go:go_default_library

