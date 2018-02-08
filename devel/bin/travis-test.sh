#!/bin/bash

# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Exit if any command returns non-zero.
set -e

# Print all commands before executing.
set -x

build_dir="$(pwd)"
install_dir="$build_dir/install_dir"

# Fetch Clownfish.
git clone -q --depth 1 https://git-wip-us.apache.org/repos/asf/lucy-clownfish.git

test_c() {
    # Install Clownfish.
    cd lucy-clownfish/compiler/c
    ./configure --prefix="$install_dir"
    make -j install
    cd ../../runtime/c
    ./configure --prefix="$install_dir"
    make -j install

    # Needed to find DLL on Windows.
    export PATH="$install_dir/bin:$PATH"

    cd ../../../c
    ./configure --clownfish-prefix "$install_dir"
    make -j test
}

test_perl() {
    # Test::Harness defaults to PERL_USE_UNSAFE_INC=1
    export PERL_USE_UNSAFE_INC=0

    source ~/perl5/perlbrew/etc/bashrc
    perlbrew switch $PERL_VERSION ||
        perlbrew install --switch --notest --noman --thread $PERL_VERSION
    perlbrew list
    export PERL5LIB="$install_dir/lib/perl5"

    # Install Clownfish.
    cd lucy-clownfish/compiler/perl
    cpanm --quiet --installdeps --notest .
    perl Build.PL
    ./Build install --install-base "$install_dir"
    cd ../../runtime/perl
    perl Build.PL
    ./Build install --install-base "$install_dir"

    cd ../../../perl
    perl Build.PL
    ./Build test
}

test_go() {
    export GOPATH="$install_dir"
    mkdir -p "$install_dir/src/git-wip-us.apache.org/repos/asf"
    ln -s "$build_dir/lucy-clownfish" \
        "$install_dir/src/git-wip-us.apache.org/repos/asf/lucy-clownfish.git"
    ln -s "$build_dir" \
        "$install_dir/src/git-wip-us.apache.org/repos/asf/lucy.git"

    # Install Clownfish.
    cd lucy-clownfish/compiler/go
    go run build.go install
    cd ../../runtime/go
    go run build.go install

    cd ../../../go
    go run build.go test
}

case $CLOWNFISH_HOST in
    perl)
        test_perl
        ;;
    c)
        test_c
        ;;
    go)
        test_go
        ;;
    *)
        echo "unknown CLOWNFISH_HOST: $CLOWNFISH_HOST"
        exit 1
esac

