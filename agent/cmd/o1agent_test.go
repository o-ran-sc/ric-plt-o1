/*
==================================================================================
  Copyright (c) 2020 AT&T Intellectual Property.
  Copyright (c) 2020 Nokia

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
==================================================================================
*/

package main

import (
	"github.com/stretchr/testify/assert"
	"os"
	"syscall"
	"testing"
	"time"
)

var o1Agent *O1Agent

// Test cases
func TestMain(M *testing.M) {
	o1Agent = NewO1Agent()
	go func() {
		o1Agent.nbiClient.Start()
		o1Agent.Run()
	}()
	time.Sleep(time.Duration(1) * time.Second)
	os.Exit(M.Run())
}

func TestConsume(t *testing.T) {
	ret := o1Agent.Consume(nil)
	assert.Nil(t, ret)
}

func TestConfigChangeHandler(t *testing.T) {
	o1Agent.ConfigChangeHandler("")
}

func TestStatusCB(t *testing.T) {
	assert.True(t, o1Agent.StatusCB())
}

func TestSighandler(t *testing.T) {
	oldOsExit := osExit
	defer func() { osExit = oldOsExit }()

	var exitCode int
	osExit = func(code int) {
		exitCode = code
	}

	go o1Agent.Sighandler()
	o1Agent.sigChan <- syscall.SIGTERM
	time.Sleep(time.Duration(1) * time.Second)
	assert.Equal(t, 1, exitCode)
}
