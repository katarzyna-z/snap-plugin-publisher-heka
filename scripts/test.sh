#!/bin/bash -e

#http://www.apache.org/licenses/LICENSE-2.0.txt
#
#
#Copyright 2015 Intel Corporation
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

# The script does automatic checking on a Go package and its sub-packages, including:
# 1. gofmt         (http://golang.org/cmd/gofmt/)
# 2. goimports     (https://github.com/bradfitz/goimports)
# 3. golint        (https://github.com/golang/lint)
# 4. go vet        (http://golang.org/cmd/vet)
# 5. race detector (http://blog.golang.org/race-detector)
# 6. test coverage (http://blog.golang.org/cover)

# Capture what test we should run
TEST_TYPE="${TEST_TYPE:-$1}"


[[ "$TEST_TYPE" =~ ^(unit|integration)$ ]] || echo "invalid/missing TEST_TYPE (value must be 'unit' or 'integration' received:${TEST_TYPE}"

__dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
__proj_dir="$(dirname "$__dir")"

# shellcheck source=scripts/common.sh
. "${__dir}/common.sh"

mozilla_path=$(echo ${GOPATH//://src/github.com/mozilla-services:}/src/github.com/mozilla-services | cut -d':' -f1)
heka_path="${mozilla_path}/heka"

# append Heka build to GOPATH
export GOPATH="${heka_path}/build/heka:${GOPATH}"
_debug "heka custom GOPATH: ${GOPATH}"

if [[ $TEST_TYPE == "unit" ]]; then
        go get github.com/axw/gocov/gocov
        go get github.com/mattn/goveralls
        go get golang.org/x/tools/cmd/goimports
        go get github.com/smartystreets/goconvey/convey
        go get golang.org/x/tools/cmd/cover

        COVERALLS_TOKEN=t47LG6BQsfLwb9WxB56hXUezvwpED6D11
        TEST_DIRS="main.go snapheka/"
        VET_DIRS=". ./snapheka/..."

        set -e

        # Automatic checks
        echo "gofmt"
        test -z "$(gofmt -l -d $TEST_DIRS | tee /dev/stderr)"

        echo "goimports"
        test -z "$(goimports -l -d $TEST_DIRS | tee /dev/stderr)"

        # Useful but should not fail on link per: https://github.com/golang/lint
        # "The suggestions made by golint are exactly that: suggestions. Golint is not perfect,
        # and has both false positives and false negatives. Do not treat its output as a gold standard.
        # We will not be adding pragmas or other knobs to suppress specific warnings, so do not expect
        # or require code to be completely "lint-free". In short, this tool is not, and will never be,
        # trustworthy enough for its suggestions to be enforced automatically, for example as part of
        # a build process"
        # echo "golint"
        # golint ./...

        echo "go vet"
        go vet $VET_DIRS
        # go test -race ./... - Lets disable for now

        # Run test coverage on each subdirectories and merge the coverage profile.
        echo "mode: count" > profile.cov

        # Standard go tooling behavior is to ignore dirs with leading underscors
        for dir in $(find . -maxdepth 10 -not -path './.git*' -not -path '*/_*' -not -path './examples/*' -not -path './scripts/*' -not -path './build/*' -not -path './Godeps/*' -type d);
        do
                if ls $dir/*.go &> /dev/null; then
                        go test --tags=unit -covermode=count -coverprofile=$dir/profile.tmp $dir
                        if [ -f $dir/profile.tmp ]
                        then
                                cat $dir/profile.tmp | tail -n +2 >> profile.cov
                                rm $dir/profile.tmp
                        fi
                fi
        done

        go tool cover -func profile.cov

elif [[ $TEST_TYPE == "integration" ]]; then
  docker images --format "{{.Repository}}" | grep "^mozilla/heka$" || (cd "${heka_path}/docker" && ./build_docker.sh)
  _info "starting heka container"
  heka_id=$(docker run -d -it -p 4352:4352 -p 3242:3242 -v "${__proj_dir}/examples/tcp-docker-test.toml":/etc/heka/config.toml mozilla/heka -config /etc/heka/config.toml)
  _debug "container id: ${heka_id}"
  _info "waiting for heka service"
  set +e
  set +u
  DOCKER_HOST="${HOST:-}"
  if [[ -z $DOCKER_HOST ]]; then
        ip=$(docker inspect -f '{{ .NetworkSettings.IPAddress }}' "${heka_id}")
  else
        ip=`echo $DOCKER_HOST | grep -o '[0-9]\+[.][0-9]\+[.][0-9]\+[.][0-9]\+'`     
  fi
  set -e
  while ! curl --silent -G "http://${ip}:4352" > /dev/null 2>&1 ; do
    sleep 1
    echo -n "."
  done
  _info "running integration tests"
  SNAP_HEKA_HOST=$ip go test -v --tags=integration ./...
  _info "cleanup heka container"
  docker stop "${heka_id}"
  docker rm "${heka_id}"
fi
