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

package sbi

import (
	"bytes"
	"fmt"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"gerrit.o-ran-sc.org/r/ric-plt/xapp-frame/pkg/xapp"
	apiclient "gerrit.oran-osc.org/r/ric-plt/o1mediator/pkg/appmgrclient"
	apixapp "gerrit.oran-osc.org/r/ric-plt/o1mediator/pkg/appmgrclient/xapp"
	apimodel "gerrit.oran-osc.org/r/ric-plt/o1mediator/pkg/appmgrmodel"
)

type PodStatus struct {
	Name   string
	Health string
	Status string
}

var log = xapp.Logger

func NewSBIClient(host, baseUrl string, prot []string, timo int) *SBIClient {
	return &SBIClient{host, baseUrl, prot, time.Duration(timo) * time.Second}
}

func (s *SBIClient) CreateTransport() *apiclient.RICAppmgr {
	return apiclient.New(httptransport.New(s.host, s.baseUrl, s.prot), strfmt.Default)
}

func (s *SBIClient) BuildXappDescriptor(name, namespace, release, version string) *apimodel.XappDescriptor {
	return &apimodel.XappDescriptor{
		XappName:    &name,
		HelmVersion: version,
		ReleaseName: release,
		Namespace:   namespace,
	}
}

func (s *SBIClient) DeployXapp(xappDesc *apimodel.XappDescriptor) error {
	params := apixapp.NewDeployXappParamsWithTimeout(s.timeout).WithXappDescriptor(xappDesc)
	log.Info("SBI: DeployXapp=%v", params)

	result, err := s.CreateTransport().Xapp.DeployXapp(params)
	if err != nil {
		log.Error("SBI: DeployXapp unsuccessful: %v", err)
	} else {
		log.Info("SBI: DeployXapp successful: payload=%v", result.Payload)
	}
	return err
}

func (s *SBIClient) UndeployXapp(xappDesc *apimodel.XappDescriptor) error {
	name := *xappDesc.XappName
	if xappDesc.ReleaseName != "" {
		name = xappDesc.ReleaseName
	}

	params := apixapp.NewUndeployXappParamsWithTimeout(s.timeout).WithXAppName(name)
	log.Info("SBI: UndeployXapp=%v", params)

	result, err := s.CreateTransport().Xapp.UndeployXapp(params)
	if err != nil {
		log.Error("SBI: UndeployXapp unsuccessful: %v", err)
	} else {
		log.Info("SBI: UndeployXapp successful: payload=%v", result)
	}
	return err
}

func (s *SBIClient) GetDeployedXapps() error {
	params := apixapp.NewGetAllXappsParamsWithTimeout(s.timeout)
	result, err := s.CreateTransport().Xapp.GetAllXapps(params)
	if err != nil {
		log.Error("GET unsuccessful: %v", err)
	} else {
		log.Info("GET successful: payload=%v", result.Payload)
	}
	return err
}

func (s *SBIClient) BuildXappConfig(name, namespace string, configData interface{}) *apimodel.XAppConfig {
	metadata := &apimodel.ConfigMetadata{
		XappName:  &name,
		Namespace: &namespace,
	}

	return &apimodel.XAppConfig{
		Metadata: metadata,
		Config:   configData,
	}
}

func (s *SBIClient) ModifyXappConfig(xappConfig *apimodel.XAppConfig) error {
	params := apixapp.NewModifyXappConfigParamsWithTimeout(s.timeout).WithXAppConfig(xappConfig)
	result, err := s.CreateTransport().Xapp.ModifyXappConfig(params)
	if err != nil {
		log.Error("SBI: ModifyXappConfig unsuccessful: %v", err)
	} else {
		log.Info("SBI: ModifyXappConfig successful: payload=%v", result.Payload)
	}
	return err
}

func (s *SBIClient) GetAllPodStatus(namespace string) ([]PodStatus, error) {
	output, err := s.RunCommand(fmt.Sprintf("kubectl get pod -n %s", namespace))
	if err != nil {
		return []PodStatus{}, err
	}

	podStatusList := []PodStatus{}
	var readyStr string
	re := regexp.MustCompile(fmt.Sprintf(`%s-.*`, namespace))
	podList := re.FindAllStringSubmatch(string(output), -1)
	if podList != nil {
		for _, pod := range podList {
			p := PodStatus{}
			fmt.Sscanf(pod[0], "%s %s %s", &p.Name, &readyStr, &p.Status)
			p.Name = strings.Split(p.Name, "-")[1]
			p.Health = s.GetHealthState(readyStr)

			podStatusList = append(podStatusList, p)
		}
	}
	return podStatusList, nil
}

func (s *SBIClient) GetHealthState(ready string) (state string) {
	result := strings.Split(ready, "/")
	if len(result) < 2 {
		return "unavailable"
	}

	if result[0] == result[1] {
		state = "healthy"
	} else {
		state = "unhealthy"
	}
	return
}

func (s *SBIClient) RunCommand(args string) (string, error) {
	return CommandExec(args)
}

var CommandExec = func(args string) (string, error) {
	cmd := exec.Command("/bin/sh", "-c", args)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	xapp.Logger.Debug("Running command: '%s'", cmd)
	if err := cmd.Run(); err != nil {
		xapp.Logger.Error("Command failed (%s): %v - %s", cmd, err.Error(), stderr.String())
		return "", err
	}
	xapp.Logger.Debug("Command executed successfully!")
	return stdout.String(), nil
}
