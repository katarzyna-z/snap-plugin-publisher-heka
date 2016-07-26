#!/bin/bash -e

#http://www.apache.org/licenses/LICENSE-2.0.txt
#
#
#Copyright 2016 Intel Corporation
#
#Licensed under the Apache License, Version 2.0 (the "License");
#you may not use this file except in compliance with the License.
#You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
#Unless required by applicable law or agreed to in writing, software
#distributed under the License is distributed on an "AS IS" BASIS,
#WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#See the License for the specific language governing permissions and
#limitations under the License.

set -e
set -u
set -o pipefail

GITVERSION=`git describe --always`

__dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
__proj_dir="$(dirname "$__dir")"

SOURCEDIR="${__proj_dir}"
BUILDDIR=$SOURCEDIR/build
PLUGIN=`echo $SOURCEDIR | grep -oh "snap-plugin-.*"`
ROOTFS=$BUILDDIR/rootfs
BUILDCMD='go build -a -ldflags "-w"'

# shellcheck source=scripts/common.sh
. "${__dir}/common.sh"

mozilla_path=$(echo ${GOPATH//://src/github.com/mozilla-services:}/src/github.com/mozilla-services | cut -d':' -f1)
heka_path="${mozilla_path}/heka"

_debug "heka path: ${heka_path}"
_info "ensure latest mozilla heka repo available"

[[ -d ${heka_path} ]] || _fail "run 'make dep' and ensure mozilla/heka source exist in ${heka_path}"

_info "building heka"
# NOTE: heka buid scripts does not honor set -e and set -u
set +e
set +u
(cd "${heka_path}" && source ./build.sh)
set -e
set -u

# append Heka build to GOPATH
export GOPATH="${heka_path}/build/heka:${GOPATH}"
_debug "heka custom GOPATH: ${GOPATH}"

_info "building snap heka plugin"
cd "${__proj_dir}"

# Disable CGO for builds
export CGO_ENABLED=0

# Clean build bin dir
rm -rf "${ROOTFS:?}"/*

# Make dir
mkdir -p "${ROOTFS}"

# Build plugin
echo "Source Dir = $SOURCEDIR"
echo "Building snap Plugin: $PLUGIN"
$BUILDCMD -o $ROOTFS/$PLUGIN
