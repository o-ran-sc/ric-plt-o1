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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"github.com/valyala/fastjson"
	"os"
	"strings"
	"time"
	"unsafe"

	"gerrit.o-ran-sc.org/r/ric-plt/xapp-frame/pkg/xapp"
	"gerrit.oran-osc.org/r/ric-plt/o1mediator/pkg/sbi"
)

/*
#cgo LDFLAGS: -lsysrepo -lyang

#include <stdio.h>
#include <limits.h>
#include <sysrepo.h>
#include <sysrepo/values.h>
#include "helper.h"
*/
import "C"

var sbiClient sbi.SBIClientInterface
var nbiClient *Nbi
var log = xapp.Logger
var rnib iRnib = xapp.Rnib

func NewNbi(s sbi.SBIClientInterface) *Nbi {
	sbiClient = s

	nbiClient = &Nbi{
		schemas:     viper.GetStringSlice("nbi.schemas"),
		cleanupChan: make(chan bool),
	}
	return nbiClient
}

func (n *Nbi) Start() bool {
	if ok := n.Setup(n.schemas); !ok {
		log.Error("NBI: SYSREPO initialization failed, bailing out!")
		return false
	}
	log.Info("NBI: SYSREPO initialization done ... processing O1 requests!")

	return true
}

func (n *Nbi) Stop() {
	C.sr_unsubscribe(n.subscription)
	C.sr_session_stop(n.session)
	C.sr_disconnect(n.connection)

	log.Info("NBI: SYSREPO cleanup done gracefully!")
}

func (n *Nbi) Setup(schemas []string) bool {
	rc := C.sr_connect(0, &n.connection)
	if C.SR_ERR_OK != rc {
		log.Error("NBI: sr_connect failed: %s", C.GoString(C.sr_strerror(rc)))
		return false
	}

	rc = C.sr_session_start(n.connection, C.SR_DS_RUNNING, &n.session)
	if C.SR_ERR_OK != rc {
		log.Error("NBI: sr_session_start failed: %s", C.GoString(C.sr_strerror(rc)))
		return false
	}

	for {
		if ok := n.DoSubscription(schemas); ok == true {
			break
		}
		time.Sleep(time.Duration(5 * time.Second))
	}
	return true
}

func (n *Nbi) DoSubscription(schemas []string) bool {
	log.Info("Subscribing YANG modules ... %v", schemas)

	for _, module := range schemas {
		modName := C.CString(module)
		defer C.free(unsafe.Pointer(modName))

		if done := n.SubscribeModule(modName); !done {
			return false
		}
	}
	return n.SubscribeStatusData()
}

func (n *Nbi) SubscribeModule(module *C.char) bool {
	rc := C.sr_module_change_subscribe(n.session, module, nil, C.sr_module_change_cb(C.module_change_cb), nil, 0, 0, &n.subscription)
	if C.SR_ERR_OK != rc {
		log.Info("NBI: sr_module_change_subscribe failed: %s", C.GoString(C.sr_strerror(rc)))
		return false
	}
	return true
}

func (n *Nbi) SubscribeStatusData() bool {
	if ok := n.SubscribeStatus("o-ran-sc-ric-gnb-status-v1", "/o-ran-sc-ric-gnb-status-v1:ric/nodes"); !ok {
		return ok
	}

	if ok := n.SubscribeStatus("o-ran-sc-ric-xapp-desc-v1", "/o-ran-sc-ric-xapp-desc-v1:ric/health"); !ok {
		return ok
	}

	if ok := n.SubscribeStatus("o-ran-sc-ric-alarm-v1", "/o-ran-sc-ric-alarm-v1:ric/alarms"); !ok {
		return ok
	}

	return true
}

func (n *Nbi) SubscribeStatus(module, xpath string) bool {
	mod := C.CString(module)
	path := C.CString(xpath)
	defer C.free(unsafe.Pointer(mod))
	defer C.free(unsafe.Pointer(path))

	rc := C.sr_oper_get_items_subscribe(n.session, mod, path, C.sr_oper_get_items_cb(C.gnb_status_cb), nil, 0, &n.subscription)
	if C.SR_ERR_OK != rc {
		log.Error("NBI: sr_oper_get_items_subscribe failed: %s", C.GoString(C.sr_strerror(rc)))
		return false
	}
	return true
}

//export nbiModuleChangeCB
func nbiModuleChangeCB(session *C.sr_session_ctx_t, module *C.char, xpath *C.char, event C.sr_event_t, reqId C.int) C.int {
	changedModule := C.GoString(module)
	changedXpath := C.GoString(xpath)

	log.Info("NBI: change event='%d' module=%s xpath=%s reqId=%d", event, changedModule, changedXpath, reqId)
	if C.SR_EV_CHANGE != event {
		log.Info("NBI: Changes finalized!")
		return C.SR_ERR_OK
	}

	if changedModule == "o-ran-sc-ric-xapp-desc-v1" {
		configJson := C.yang_data_sr2json(session, module, event, &nbiClient.oper)
		err := nbiClient.ManageXapps(changedModule, C.GoString(configJson), int(nbiClient.oper))
		if err != nil {
			return C.SR_ERR_OPERATION_FAILED
		}
	}

	if changedModule == "o-ran-sc-ric-ueec-config-v1" {
		configJson := C.get_data_json(session, module)
		err := nbiClient.ManageConfigmaps(changedModule, C.GoString(configJson), int(C.SR_OP_MODIFIED))
		if err != nil {
			return C.SR_ERR_OPERATION_FAILED
		}
	}

	return C.SR_ERR_OK
}

func (n *Nbi) ManageXapps(module, configJson string, oper int) error {
	log.Info("ManageXapps: module=%s configJson=%s", module, configJson)

	if configJson == "" {
		return nil
	}

	root := fmt.Sprintf("%s:ric", module)
	jsonList, err := n.ParseJsonArray(configJson, root, "xapps", "xapp")
	if err != nil {
		return err
	}

	for _, m := range jsonList {
		xappName := string(m.GetStringBytes("name"))
		namespace := string(m.GetStringBytes("namespace"))
		relName := string(m.GetStringBytes("release-name"))
		version := string(m.GetStringBytes("version"))

		desc := sbiClient.BuildXappDescriptor(xappName, namespace, relName, version)
		switch oper {
		case C.SR_OP_CREATED:
			return sbiClient.DeployXapp(desc)
		case C.SR_OP_DELETED:
			return sbiClient.UndeployXapp(desc)
		default:
			return errors.New(fmt.Sprintf("Operation '%d' not supported!", oper))
		}
	}
	return nil
}

func (n *Nbi) ManageConfigmaps(module, configJson string, oper int) error {
	log.Info("ManageConfig: module=%s configJson=%s", module, configJson)

	if configJson == "" {
		return nil
	}

	if oper != C.SR_OP_MODIFIED {
		return errors.New(fmt.Sprintf("Operation '%d' not supported!", oper))
	}

	value, err := n.ParseJson(configJson)
	if err != nil {
		log.Info("ParseJson failed with error: %v", oper)
		return err
	}

	root := fmt.Sprintf("%s:ric", module)
	appName := string(value.GetStringBytes(root, "config", "name"))
	namespace := string(value.GetStringBytes(root, "config", "namespace"))
	controlVal := value.Get(root, "config", "control")
	if controlVal == nil {
		return nil
	}
	control := controlVal.String()

	var f interface{}
	err = json.Unmarshal([]byte(strings.ReplaceAll(control, "\\", "")), &f)
	if err != nil {
		log.Info("json.Unmarshal failed: %v", err)
		return err
	}

	xappConfig := sbiClient.BuildXappConfig(appName, namespace, f)
	return sbiClient.ModifyXappConfig(xappConfig)
}

func (n *Nbi) ParseJson(dsContent string) (*fastjson.Value, error) {
	var p fastjson.Parser
	v, err := p.Parse(dsContent)
	if err != nil {
		log.Info("fastjson.Parser failed: %v", err)
	}
	return v, err
}

func (n *Nbi) ParseJsonArray(dsContent, model, top, elem string) ([]*fastjson.Value, error) {
	v, err := n.ParseJson(dsContent)
	if err != nil {
		return nil, err
	}
	return v.GetArray(model, top, elem), nil
}

//export nbiGnbStateCB
func nbiGnbStateCB(session *C.sr_session_ctx_t, module *C.char, xpath *C.char, rpath *C.char, reqid C.uint32_t, parent **C.char) C.int {
	mod := C.GoString(module)
	log.Info("nbiGnbStateCB: module='%s' xpath='%s' rpath='%s' [id=%d]", mod, C.GoString(xpath), C.GoString(rpath), reqid)

	if mod == "o-ran-sc-ric-xapp-desc-v1" {
		xappnamespace := os.Getenv("XAPP_NAMESPACE")
		if xappnamespace == "" {
			xappnamespace = "ricxapp"
		}
		podList, _ := sbiClient.GetAllPodStatus(xappnamespace)

		for _, pod := range podList {
			path := fmt.Sprintf("/o-ran-sc-ric-xapp-desc-v1:ric/health/status[name='%s']", pod.Name)
			nbiClient.CreateNewElement(session, parent, path, "name", path)
			nbiClient.CreateNewElement(session, parent, path, "health", pod.Health)
			nbiClient.CreateNewElement(session, parent, path, "status", pod.Status)
		}
		return C.SR_ERR_OK
	}

	if mod == "o-ran-sc-ric-alarm-v1" {
		if alerts, _ := sbiClient.GetAlerts(); alerts != nil {
			for _, alert := range alerts.Payload {
				id := alert.Annotations["alarm_id"]
				path := fmt.Sprintf("/o-ran-sc-ric-alarm-v1:ric/alarms/alarm[alarm-id='%s']", id)
				nbiClient.CreateNewElement(session, parent, path, "alarm-id", id)
				nbiClient.CreateNewElement(session, parent, path, "fault-text", alert.Alert.Labels["alertname"])
				nbiClient.CreateNewElement(session, parent, path, "severity", alert.Alert.Labels["severity"])
				nbiClient.CreateNewElement(session, parent, path, "status", alert.Alert.Labels["status"])
				nbiClient.CreateNewElement(session, parent, path, "additional-info", alert.Annotations["additional_info"])
			}
		}
		return C.SR_ERR_OK
	}

	gnbs, err := rnib.GetListGnbIds()
	if err != nil || len(gnbs) == 0 || len(gnbs) > 0 {
		//Check if any Enbs are connected to RIC
		log.Info("Rnib.GetListGnbIds() returned elementCount=%d err:%v", len(gnbs), err)
		enbs, err2 := rnib.GetListEnbIds()
			if err2 != nil || len(enbs) == 0 {
				log.Info("Rnib.GetListEnbIds() returned elementCount=%d err:%v", len(gnbs), err2)
				log.Error("No Gnbs or Enbs Connected to Ric")
				return C.SR_ERR_OK
			} else {
				log.Info("Getting Enb details from list of Enbs")
				for _, enb := range enbs {
					ranName := enb.GetInventoryName()
					info, err := rnib.GetNodeb(ranName)
					if err != nil {
						log.Error("GetNodeb() failed for ranName=%s: %v", ranName, err)
						continue
					}

					prot := nbiClient.E2APProt2Str(int(info.E2ApplicationProtocol))
					connStat := nbiClient.ConnStatus2Str(int(info.ConnectionStatus))
					ntype := nbiClient.NodeType2Str(int(info.NodeType))

					log.Info("eNB info: %s -> %s %s %s -> %s %s", ranName, prot, connStat, ntype, enb.GetGlobalNbId().GetPlmnId(), enb.GetGlobalNbId().GetNbId())

					path := fmt.Sprintf("/o-ran-sc-ric-gnb-status-v1:ric/nodes/node[ran-name='%s']", ranName)
					nbiClient.CreateNewElement(session, parent, path, "ran-name", ranName)
					nbiClient.CreateNewElement(session, parent, path, "ip", info.Ip)
					nbiClient.CreateNewElement(session, parent, path, "port", fmt.Sprintf("%d", info.Port))
					nbiClient.CreateNewElement(session, parent, path, "plmn-id", enb.GetGlobalNbId().GetPlmnId())
					nbiClient.CreateNewElement(session, parent, path, "nb-id", enb.GetGlobalNbId().GetNbId())
					nbiClient.CreateNewElement(session, parent, path, "e2ap-protocol", prot)
					nbiClient.CreateNewElement(session, parent, path, "connection-status", connStat)
					nbiClient.CreateNewElement(session, parent, path, "node", ntype)
				}
			}
	}

	for _, gnb := range gnbs {
		ranName := gnb.GetInventoryName()
		info, err := rnib.GetNodeb(ranName)
		if err != nil {
			log.Error("GetNodeb() failed for ranName=%s: %v", ranName, err)
			continue
		}

		prot := nbiClient.E2APProt2Str(int(info.E2ApplicationProtocol))
		connStat := nbiClient.ConnStatus2Str(int(info.ConnectionStatus))
		ntype := nbiClient.NodeType2Str(int(info.NodeType))

		log.Info("gNB info: %s -> %s %s %s -> %s %s", ranName, prot, connStat, ntype, gnb.GetGlobalNbId().GetPlmnId(), gnb.GetGlobalNbId().GetNbId())

		path := fmt.Sprintf("/o-ran-sc-ric-gnb-status-v1:ric/nodes/node[ran-name='%s']", ranName)
		nbiClient.CreateNewElement(session, parent, path, "ran-name", ranName)
		nbiClient.CreateNewElement(session, parent, path, "ip", info.Ip)
		nbiClient.CreateNewElement(session, parent, path, "port", fmt.Sprintf("%d", info.Port))
		nbiClient.CreateNewElement(session, parent, path, "plmn-id", gnb.GetGlobalNbId().GetPlmnId())
		nbiClient.CreateNewElement(session, parent, path, "nb-id", gnb.GetGlobalNbId().GetNbId())
		nbiClient.CreateNewElement(session, parent, path, "e2ap-protocol", prot)
		nbiClient.CreateNewElement(session, parent, path, "connection-status", connStat)
		nbiClient.CreateNewElement(session, parent, path, "node", ntype)
	}
	return C.SR_ERR_OK
}

func (n *Nbi) CreateNewElement(session *C.sr_session_ctx_t, parent **C.char, key, name, value string) {
	basePath := fmt.Sprintf("%s/%s", key, name)
	log.Info("%s -> %s", basePath, value)

	cPath := C.CString(basePath)
	defer C.free(unsafe.Pointer(cPath))
	cValue := C.CString(value)
	defer C.free(unsafe.Pointer(cValue))

	C.create_new_path(session, parent, cPath, cValue)
}

func (n *Nbi) ConnStatus2Str(connStatus int) string {
	switch connStatus {
	case 0:
		return "not-specified"
	case 1:
		return "connected"
	case 2:
		return "disconnected"
	case 3:
		return "setup-failed"
	case 4:
		return "connecting"
	case 5:
		return "shutting-down"
	case 6:
		return "shutdown"
	}
	return "not-specified"
}

func (n *Nbi) E2APProt2Str(prot int) string {
	switch prot {
	case 0:
		return "not-specified"
	case 1:
		return "x2-setup-request"
	case 2:
		return "endc-x2-setup-request"
	}
	return "not-specified"
}

func (n *Nbi) NodeType2Str(ntype int) string {
	switch ntype {
	case 0:
		return "not-specified"
	case 1:
		return "enb"
	case 2:
		return "gnb"
	}
	return "not-specified"
}

func (n *Nbi) testModuleChangeCB(module string) bool {
	var event C.sr_event_t = C.SR_EV_CHANGE
	reqID := C.int(100)
	modName := C.CString(module)
	defer C.free(unsafe.Pointer(modName))

	if ret := nbiModuleChangeCB(n.session, modName, nil, event, reqID); ret != C.SR_ERR_OK {
		return false
	}
	return true
}

func (n *Nbi) testModuleChangeCBDone(module string) bool {
	var event C.sr_event_t = C.SR_EV_DONE
	reqID := C.int(100)
	modName := C.CString(module)
	defer C.free(unsafe.Pointer(modName))

	if ret := nbiModuleChangeCB(n.session, modName, nil, event, reqID); ret != C.SR_ERR_OK {
		return false
	}
	return true
}

func (n *Nbi) testGnbStateCB(module string) bool {
	modName := C.CString(module)
	defer C.free(unsafe.Pointer(modName))
	reqID := C.uint32_t(100)
	parent := make([]*C.char, 1)

	if ret := nbiGnbStateCB(n.session, modName, nil, nil, reqID, &parent[0]); ret != C.SR_ERR_OK {
		return false
	}
	return true
}

type iRnib interface {
	GetListGnbIds() ([]*xapp.RNIBNbIdentity, xapp.RNIBIRNibError)
	GetListEnbIds() ([]*xapp.RNIBNbIdentity, xapp.RNIBIRNibError)
	GetNodeb(invName string) (*xapp.RNIBNodebInfo, xapp.RNIBIRNibError)
}
