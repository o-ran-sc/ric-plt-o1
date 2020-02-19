/*
==================================================================================
  Copyright (c) 2019 AT&T Intellectual Property.
  Copyright (c) 2019 Nokia

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
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"gerrit.o-ran-sc.org/r/ric-plt/xapp-frame/pkg/xapp"
	"gerrit.oran-osc.org/r/ric-plt/o1mediator/pkg/nbi"
	"gerrit.oran-osc.org/r/ric-plt/o1mediator/pkg/sbi"
	"github.com/spf13/viper"
)

var Version string
var Hash string

type O1Agent struct {
	rmrReady  bool
	nbiClient *nbi.Nbi
	sigChan   chan os.Signal
}

func (o O1Agent) Consume(rp *xapp.RMRParams) (err error) {
	xapp.Logger.Debug("Message received!")
	return nil
}

func (o *O1Agent) ConfigChangeHandler(f string) {
	xapp.Logger.Debug("Config changed!")
}

func (o *O1Agent) StatusCB() bool {
	if !o.rmrReady {
		xapp.Logger.Info("RMR not ready yet!")
	}
	return true
}

func (o *O1Agent) Run() {
	xapp.Logger.SetMdc("o1agent", fmt.Sprintf("%s:%s", Version, Hash))
	xapp.SetReadyCB(func(d interface{}) { o.rmrReady = true }, true)
	xapp.AddConfigChangeListener(o.ConfigChangeHandler)
	xapp.Resource.InjectStatusCb(o.StatusCB)

	signal.Notify(o.sigChan, syscall.SIGINT, syscall.SIGTERM)
	go o.Sighandler()

	xapp.Run(o)
}

func (o *O1Agent) Sighandler() {
	xapp.Logger.Info("Signal handler installed!")

	<-o.sigChan
	o.nbiClient.Stop()
	os.Exit(1)
}

func NewO1Agent() *O1Agent {
	host := viper.GetString("sbi.appmgrService")
	baseUrl := viper.GetString("sbi.baseUrl")
	prot := viper.GetString("sbi.proto")
	timeout := viper.GetInt("sbi.timeout")

	sbiClient := sbi.NewSBIClient(host, baseUrl, []string{prot}, timeout)

	return &O1Agent{
		rmrReady:  false,
		nbiClient: nbi.NewNbi(sbiClient),
		sigChan:   make(chan os.Signal, 1),
	}
}

func main() {
	o1Agent := NewO1Agent()

	if ok := o1Agent.nbiClient.Start(); !ok {
		xapp.Logger.Error("NBI initialization failed!")
		return
	}

	o1Agent.Run()
}
