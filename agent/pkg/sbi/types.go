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
	"time"

	apimodel "gerrit.oran-osc.org/r/ric-plt/o1mediator/pkg/appmgrmodel"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
)

type SBIClient struct {
	appmgrAddr   string
	alertmgrAddr string
	timeout 	 time.Duration
}

type SBIClientInterface interface {
	BuildXappDescriptor(name, version, release, namespace string) *apimodel.XappDescriptor
	DeployXapp(xappDesc *apimodel.XappDescriptor) error
	UndeployXapp(xappDesc *apimodel.XappDescriptor) error
	GetDeployedXapps() error

	BuildXappConfig(name, namespace string, configData interface{}) *apimodel.XAppConfig
	ModifyXappConfig(xappConfig *apimodel.XAppConfig) error

	GetAllPodStatus(namespace string) ([]PodStatus, error)

	GetAlerts() (*alert.GetAlertsOK, error)
}
