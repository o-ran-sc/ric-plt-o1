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

package sbi_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-openapi/strfmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	apimodel "gerrit.oran-osc.org/r/ric-plt/o1mediator/pkg/appmgrmodel"
	"gerrit.oran-osc.org/r/ric-plt/o1mediator/pkg/sbi"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/stretchr/testify/assert"
)

var s *sbi.SBIClient

var xappName = "ueec"
var ns = "ricxapp"
var release = "ueec-xapp"
var helmVer = "0.0.1"
var cfgData = `{
	"active":true,
	"interfaceId": {
		"globalENBId":{
			"plmnId": "1234",
			"eNBId":"55"
		}
	}
}`
var kpodOutput = `
NAME                               READY   STATUS    RESTARTS   AGE
ricxapp-ueec-7bfdd587db-2jl9j      1/1     Running   53         29d
ricxapp-anr-6748846478-8hmtz       1-1     Running   1          29d
ricxapp-dualco-7f76f65c99-5p6c6    0/1     Running   1          29d
`

// Test cases
func TestMain(M *testing.M) {
	s = sbi.NewSBIClient("localhost:8080", "localhost:9093", 5)
	os.Exit(M.Run())
}

func TestBuildXappDescriptor(t *testing.T) {
	var expDesc apimodel.XappDescriptor = apimodel.XappDescriptor{
		XappName:    &xappName,
		HelmVersion: helmVer,
		ReleaseName: release,
		Namespace:   ns,
	}
	assert.Equal(t, expDesc, *getTestXappDescriptor())
}

func TestDeployXapp(t *testing.T) {
	ts := createHTTPServer(t, "POST", "/ric/v1/xapps", 8080, http.StatusCreated, apimodel.Xapp{})
	defer ts.Close()
	err := s.DeployXapp(getTestXappDescriptor())
	assert.Nil(t, err)
}

func TestDeployXappReturnsErrorIfHttpErrorResponse(t *testing.T) {
	ts := createHTTPServer(t, "POST", "/ric/v1/xapps", 8080, http.StatusInternalServerError, nil)
	defer ts.Close()
	err := s.DeployXapp(getTestXappDescriptor())
	assert.NotNil(t, err)
}

func TestUndeployXapp(t *testing.T) {
	ts := createHTTPServer(t, "DELETE", "/ric/v1/xapps/ueec-xapp", 8080, http.StatusNoContent, nil)
	defer ts.Close()
	err := s.UndeployXapp(getTestXappDescriptor())
	assert.Nil(t, err)
}

func TestUndeployXappReturnsErrorIfHttpErrorResponse(t *testing.T) {
	ts := createHTTPServer(t, "DELETE", "/ric/v1/xapps/ueec-xapp", 8080, http.StatusInternalServerError, nil)
	defer ts.Close()
	err := s.UndeployXapp(getTestXappDescriptor())
	assert.NotNil(t, err)
}

func TestGetDeployedXapps(t *testing.T) {
	ts := createHTTPServer(t, "GET", "/ric/v1/xapps", 8080, http.StatusOK, apimodel.AllDeployedXapps{})
	defer ts.Close()
	err := s.GetDeployedXapps()
	assert.Nil(t, err)
}

func TestGetDeployedXappsReturnsErrorIfHttpErrorResponse(t *testing.T) {
	ts := createHTTPServer(t, "GET", "/ric/v1/xapps", 8080, http.StatusInternalServerError, apimodel.AllDeployedXapps{})
	defer ts.Close()
	err := s.GetDeployedXapps()
	assert.NotNil(t, err)
}

func TestBuildXappConfig(t *testing.T) {
	expResp := &apimodel.XAppConfig{
		Metadata: &apimodel.ConfigMetadata{
			XappName:  &xappName,
			Namespace: &ns,
		},
		Config: cfgData,
	}
	resp := s.BuildXappConfig(xappName, ns, cfgData)
	assert.Equal(t, expResp, resp)
}

func TestModifyXappConfig(t *testing.T) {
	ts := createHTTPServer(t, "PUT", "/ric/v1/config", 8080, http.StatusOK, apimodel.ConfigValidationErrors{})
	defer ts.Close()
	xappCfg := s.BuildXappConfig(xappName, ns, cfgData)
	err := s.ModifyXappConfig(xappCfg)
	assert.Nil(t, err)
}

func TestModifyXappConfigReturnsErrorIfHttpErrorResponse(t *testing.T) {
	ts := createHTTPServer(t, "PUT", "/ric/v1/config", 8080, http.StatusInternalServerError, nil)
	defer ts.Close()
	xappCfg := s.BuildXappConfig(xappName, ns, cfgData)
	err := s.ModifyXappConfig(xappCfg)
	assert.NotNil(t, err)
}

func TestGetAllPodStatus(t *testing.T) {
	oldCmdExec := sbi.CommandExec
	defer func() { sbi.CommandExec = oldCmdExec }()
	sbi.CommandExec = func(args string) (out string, err error) {
		assert.Equal(t, "/usr/local/bin/kubectl get pod -n ricxapp", args)
		return kpodOutput, nil
	}

	expPodList := []sbi.PodStatus{
		sbi.PodStatus{
			Name:   "ueec",
			Health: "healthy",
			Status: "Running",
		},
		sbi.PodStatus{
			Name:   "anr",
			Health: "unavailable",
			Status: "Running",
		},
		sbi.PodStatus{
			Name:   "dualco",
			Health: "unhealthy",
			Status: "Running",
		},
	}

	podList, err := s.GetAllPodStatus("ricxapp")
	assert.Nil(t, err)
	assert.Equal(t, podList, expPodList)
}

func TestGetAllPodStatusReturnsErrorIfKubectlCommandFails(t *testing.T) {
	oldCmdExec := sbi.CommandExec
	defer func() { sbi.CommandExec = oldCmdExec }()
	sbi.CommandExec = func(args string) (out string, err error) {
		assert.Equal(t, "/usr/local/bin/kubectl get pod -n ricxapp", args)
		return "", errors.New("Some kubectl error")
	}

	expPodList := make([]sbi.PodStatus, 0)

	podList, err := s.GetAllPodStatus("ricxapp")
	assert.NotNil(t, err)
	assert.Equal(t, podList, expPodList)
}

func TestGetHealthState(t *testing.T) {
	assert.Equal(t, "healthy", s.GetHealthState("1/1"))
}

func TestGetHealthStateReturnsUnavailableIfReadyStatusUnkown(t *testing.T) {
	assert.Equal(t, "unavailable", s.GetHealthState(""))
}

func TestGetHealthStateReturnsUnhealthyIfReadyStatusNotUp(t *testing.T) {
	assert.Equal(t, "unhealthy", s.GetHealthState("0/1"))
}

func TestGetAlerts(t *testing.T) {
	tim := strfmt.DateTime(time.Now())
	fingerprint := "34c8f717936f063f"

	alerts := []models.GettableAlert{
		models.GettableAlert{
			Alert: models.Alert{
				Labels: models.LabelSet{
					"status":      "active",
					"alertname":   "E2 CONNECTIVITY LOST TO G-NODEB",
					"severity":    "MAJOR",
					"service":     "RIC:UEEC",
					"system_name": "RIC",
				},
			},
			Annotations: models.LabelSet{
				"alarm_id":        "8006",
				"additional_info": "ethernet",
				"description":     "eth12",
				"instructions":    "Not defined",
				"summary":         "Communication error",
			},
			EndsAt:      &tim,
			StartsAt:    &tim,
			UpdatedAt:   &tim,
			Fingerprint: &fingerprint,
		},
	}

	url := "/api/v2/alerts?active=true&inhibited=true&silenced=true&unprocessed=true"
	ts := createHTTPServer(t, "GET", url, 9093, http.StatusOK, alerts)
	defer ts.Close()

	resp, err := s.GetAlerts()

	assert.Nil(t, err)
	assert.Equal(t, true, resp != nil)
	assert.NotNil(t, resp)

	for _, alert := range resp.Payload {
		assert.Equal(t, alert.Annotations, alerts[0].Annotations)
		assert.Equal(t, alert.Alert, alerts[0].Alert)
		assert.Equal(t, alert.Fingerprint, alerts[0].Fingerprint)
	}
}

func TestGetAlertsReturnErrorIfHttpErrorResponse(t *testing.T) {
	url := "/api/v2/alerts?active=true&inhibited=true&silenced=true&unprocessed=true"
	ts := createHTTPServer(t, "GET", url, 9093, http.StatusInternalServerError, nil)
	defer ts.Close()

	resp, err := s.GetAlerts()
	assert.NotNil(t, err)
	assert.Nil(t, resp)
}

func TestCommandExec(t *testing.T) {
	resp, err := sbi.CommandExec("date")
	assert.NotEqual(t, "", resp)
	assert.Nil(t, err)
}

func TestCommandExecReturnErrorIfCmdExecutionFails(t *testing.T) {
	resp, err := sbi.CommandExec("some-not-existing-command")
	assert.Equal(t, "", resp)
	assert.NotNil(t, err)
}

func getTestXappDescriptor() *apimodel.XappDescriptor {
	return s.BuildXappDescriptor(xappName, ns, release, helmVer)
}

func createHTTPServer(t *testing.T, method, url string, port, status int, respData interface{}) *httptest.Server {
	l, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Error("Failed to create listener: " + err.Error())
	}
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, method)
		assert.Equal(t, r.URL.String(), url)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(status)
		b, _ := json.Marshal(respData)
		w.Write(b)
	}))
	ts.Listener.Close()
	ts.Listener = l

	ts.Start()

	return ts
}
