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

#ifndef HELPER_H
#define HELPER_H

#include <sysrepo.h>

sr_val_t *get_val(sr_val_t *val, size_t i);

int module_change_cb(sr_session_ctx_t *session, const char *module_name, const char *xpath, sr_event_t event, uint32_t request_id, void *private_data);

char * yang_data_sr2json(sr_session_ctx_t *session, const char *module_name, sr_event_t event, sr_change_oper_t *operation);

char * get_data_json(sr_session_ctx_t *session, const char *module_name);

#endif
