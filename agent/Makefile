
#   Copyright (c) 2019 AT&T Intellectual Property.
#   Copyright (c) 2019 Nokia.
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#       http://www.apache.org/licenses/LICENSE-2.0
#
#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.

.DEFAULT: go-build

default: go-build

build: go-build

test: go-test

#------------------------------------------------------------------------------
#
#------------------------------------------------------------------------------
ROOT_DIR:=$(dir $(abspath $(lastword $(MAKEFILE_LIST))))
CACHE_DIR?=$(abspath $(ROOT_DIR)/cache)

#------------------------------------------------------------------------------
#
# Build and test targets
#
#------------------------------------------------------------------------------

XAPP_NAME:=o1agent
XAPP_ROOT:=cmd
include build/make.go.mk 

XAPP_TESTENV:=RMR_SEED_RT=config/uta_rtg_ut.rt CFG_FILE=config/config-file.json
XAPP_BUILDDEPS:=
