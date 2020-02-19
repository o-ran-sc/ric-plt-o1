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

package nbi

import (
	"os"	
	"time"
	"encoding/json"
	"testing"
	"net"
    "net/http"
    "net/http/httptest"
	"github.com/stretchr/testify/assert"

	apimodel "gerrit.oran-osc.org/r/ric-plt/o1mediator/pkg/appmgrmodel"
	"gerrit.oran-osc.org/r/ric-plt/o1mediator/pkg/sbi"
	
)

var XappConfig = `{
	"o-ran-sc-ric-ueec-config-v1:ric": {
	  "config": {
		"name": "ueec",
		"namespace": "ricxapp",
		"control": {
		  "active": true,
		  "interfaceId": {
			"globalENBId": {
			  "plmnId": "1234",
			  "eNBId": "55"
			}
		  }
		}
	  }
	}
  }`

var XappDescriptor = `{
	"o-ran-sc-ric-xapp-desc-v1:ric": {
	  "xapps": {
		"xapp": [
		  {
			"name": "ueec",
			"release-name": "ueec-xapp",
			"version": "0.0.1",
			"namespace": "ricxapp"
		  }
		]
	  }
	}
  }`

var kpodOutput = `
NAME                                            READY   STATUS    RESTARTS   AGE
deployment-ricplt-a1mediator-7bfdd587db-2jl9j   1/1     Running   53         29d
deployment-ricplt-appmgr-6748846478-8hmtz       1-1     Running   1          29d
deployment-ricplt-dbaas-7f76f65c99-5p6c6        0/1     Running   1          29d
deployment-ricplt-e2mgr-6567bf49c7-lljl5        1/1     Running   2          29d
deployment-ricplt-e2term-76f95c8d49-6pxlm       1/1     Running   3          29d
deployment-ricplt-rtmgr-75b4886445-jsbx9        0/1     Running   1          29d
deployment-ricplt-submgr-b6b4f6c46-fclkp        1/1     Running   3          29d
deployment-ricplt-vespamgr-58f8f5659-pwbhr      1/1     Running   1          29d
o1mediator-6c7df787fb-ztb7g                     0/1     Running   0          2d
`

var n *Nbi

// Test cases
func TestMain(M *testing.M) {
	n = NewNbi(sbi.NewSBIClient("localhost:8080", "/ric/v1/", []string{"http"}, 5))
	go n.Start()
	time.Sleep(time.Duration(1) * time.Second)

	os.Exit(M.Run())
}

func TestModifyConfigmap(t *testing.T) {
	ts := CreateHTTPServer(t, "PUT", "/ric/v1/config", http.StatusOK, apimodel.ConfigValidationErrors{})
	defer ts.Close()

	var f interface{}
	err := json.Unmarshal([]byte(XappConfig), &f)
	assert.Equal(t, true, err == nil)

	err = n.ManageConfigmaps("o-ran-sc-ric-ueec-config-v1", XappConfig, 1)
	assert.Equal(t, true, err == nil)
}

func TestDeployXApp(t *testing.T) {
	ts := CreateHTTPServer(t, "POST", "/ric/v1/xapps", http.StatusCreated, apimodel.Xapp{})
	defer ts.Close()

	var f interface{}
	err := json.Unmarshal([]byte(XappDescriptor), &f)
	assert.Equal(t, true, err == nil)

	err = n.ManageXapps("o-ran-sc-ric-xapp-desc-v1", XappDescriptor, 0)
	assert.Equal(t, true, err == nil)
}

func TestUnDeployXApp(t *testing.T) {
	ts := CreateHTTPServer(t, "DELETE", "/ric/v1/xapps/ueec-xapp", http.StatusNoContent, apimodel.Xapp{})
	defer ts.Close()

	var f interface{}
	err := json.Unmarshal([]byte(XappDescriptor), &f)
	assert.Equal(t, true, err == nil)

	err = n.ManageXapps("o-ran-sc-ric-xapp-desc-v1", XappDescriptor, 2)
	assert.Equal(t, true, err == nil)
}

func TestGetDeployedXapps(t *testing.T) {
	ts := CreateHTTPServer(t, "GET", "/ric/v1/xapps", http.StatusOK, apimodel.AllDeployedXapps{})
	defer ts.Close()

	err := sbiClient.GetDeployedXapps()
	assert.Equal(t, true, err == nil)
}

func TestGetAllPodStatus(t *testing.T) {
	sbi.CommandExec = func(args string) (out string, err error) {
		assert.Equal(t, "kubectl get pod -n namespace", args)
		return kpodOutput, nil
	}

	expectedPodList := []sbi.PodStatus{
		sbi.PodStatus{Name:"a1mediator", Health: "healthy", Status:"Running"},
		sbi.PodStatus{Name:"appmgr", Health: "unavailable", Status:"Running"},
		sbi.PodStatus{Name:"dbaas", Health: "unhealthy", Status:"Running"},
		sbi.PodStatus{Name:"e2mgr", Health: "healthy", Status:"Running"},
		sbi.PodStatus{Name:"e2term", Health: "healthy", Status:"Running"},
		sbi.PodStatus{Name:"rtmgr", Health: "unhealthy", Status:"Running"},
		sbi.PodStatus{Name:"submgr", Health: "healthy", Status:"Running"},
		sbi.PodStatus{Name:"vespamgr", Health: "healthy", Status:"Running"},
	}

	podList, err := sbiClient.GetAllPodStatus("namespace")
	assert.Equal(t, true, err == nil)
	assert.Equal(t, podList, expectedPodList)
}

func TestErrorCases(t *testing.T) {
	// Invalid config
	err := n.ManageXapps("o-ran-sc-ric-xapp-desc-v1", "", 2)
	assert.Equal(t, true, err == nil)

	// Invalid module
	err = n.ManageXapps("", "{}", 2)
	assert.Equal(t, true, err == nil)

	// Unexpected module
	err = n.ManageXapps("o-ran-sc-ric-ueec-config-v1", "{}", 2)
	assert.Equal(t, true, err == nil)

	// Invalid operation
	err = n.ManageXapps("o-ran-sc-ric-ueec-config-v1", XappDescriptor, 1)
	assert.Equal(t, true, err == nil)

	// Invalid config
	err = n.ManageConfigmaps("o-ran-sc-ric-ueec-config-v1", "", 1)
	assert.Equal(t, true, err == nil)

	// Invalid module
	err = n.ManageConfigmaps("", "{}", 1)
	assert.Equal(t, true, err == nil)

	// Unexpected module
	err = n.ManageConfigmaps("o-ran-sc-ric-xapp-desc-v1", "{}", 0)
	assert.Equal(t, true, err == nil)

	// Invalid operation
	err = n.ManageConfigmaps("o-ran-sc-ric-ueec-config-v1", "{}", 0)
	assert.Equal(t, true, err != nil)
}

func TestConnStatus2Str(t *testing.T) {
	assert.Equal(t, n.ConnStatus2Str(0), "not-specified")
	assert.Equal(t, n.ConnStatus2Str(1), "connected")
	assert.Equal(t, n.ConnStatus2Str(2), "disconnected")
	assert.Equal(t, n.ConnStatus2Str(3), "setup-failed")
	assert.Equal(t, n.ConnStatus2Str(4), "connecting")
	assert.Equal(t, n.ConnStatus2Str(5), "shutting-down")
	assert.Equal(t, n.ConnStatus2Str(6), "shutdown")
	assert.Equal(t, n.ConnStatus2Str(1234), "not-specified")
}

func TestE2APProt2Str(t *testing.T) {
	assert.Equal(t, n.E2APProt2Str(0), "not-specified")
	assert.Equal(t, n.E2APProt2Str(1), "x2-setup-request")
	assert.Equal(t, n.E2APProt2Str(2), "endc-x2-setup-request")
	assert.Equal(t, n.E2APProt2Str(1111), "not-specified")
}

func TestNodeType2Str(t *testing.T) {
	assert.Equal(t, n.NodeType2Str(0), "not-specified")
	assert.Equal(t, n.NodeType2Str(1), "enb")
	assert.Equal(t, n.NodeType2Str(2), "gnb")
	assert.Equal(t, n.NodeType2Str(1111), "not-specified")
}

func TestTeardown(t *testing.T) {
	n.Stop()
}

func CreateHTTPServer(t *testing.T, method, url string, status int, respData interface{}) *httptest.Server {
	l, err := net.Listen("tcp", "localhost:8080")
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

func DescMatcher(result, expected *apimodel.XappDescriptor) bool {
	if *result.XappName == *expected.XappName && result.HelmVersion == expected.HelmVersion &&
		result.Namespace == expected.Namespace && result.ReleaseName == expected.ReleaseName {
		return true
	}
	return false
}

func ConfigMatcher(result, expected *apimodel.XAppConfig) bool {
	if *result.Metadata.XappName == *expected.Metadata.XappName &&
	   *result.Metadata.Namespace == *expected.Metadata.Namespace {
		return true
	}
	return false
}
