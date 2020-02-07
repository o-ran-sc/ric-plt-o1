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

/*
#cgo LDFLAGS: -lsysrepo

#include <sysrepo.h>
#include <sysrepo/values.h>
*/
import "C"

type Nbi struct {
	schemas      []string
	connection   *C.sr_conn_ctx_t
	session      *C.sr_session_ctx_t
  subscription *C.sr_subscription_ctx_t
  oper          C.sr_change_oper_t
	cleanupChan  chan bool
}
