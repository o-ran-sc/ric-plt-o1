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

#include <string.h>
#include "helper.h"
#include "_cgo_export.h"
#include <sysrepo.h>
#include <libyang/libyang.h>


int module_change_cb(sr_session_ctx_t *session, const char *module_name, const char *xpath, sr_event_t event, uint32_t req_id, void *priv_ctx) {
    return nbiModuleChangeCB(session, (char *)module_name, (char *)xpath, event, (int)req_id);
}

char * yang_data_sr2json(sr_session_ctx_t *session, const char *module_name, sr_event_t event, sr_change_oper_t *operation) {
    sr_change_iter_t *it = NULL;
    int rc = SR_ERR_OK;
    sr_change_oper_t oper;
    sr_val_t *old_value = NULL;
    sr_val_t *new_value = NULL;
    sr_val_t *val = NULL;
    char *json_str = NULL;
    struct lyd_node *root = NULL;

    rc = sr_get_changes_iter(session, "//." , &it);
    if (rc != SR_ERR_OK) {
        goto cleanup;
    }

    while ((rc = sr_get_change_next(session, it, &oper, &old_value, &new_value)) == SR_ERR_OK) {
        *operation = oper;
        val = oper == SR_OP_CREATED ? new_value : old_value;
        char *leaf = sr_val_to_str(val);
        if (root) {
            lyd_new_path(root, NULL, val->xpath, (void *) leaf, 0, 1);
        } else {
            root = lyd_new_path(NULL, sr_get_context(sr_session_get_connection(session)), val->xpath, (void *) leaf, 0, 1);
        }

        sr_free_val(old_value);
        sr_free_val(new_value);
        if (leaf) {
            free(leaf);
        }
    }

    if (root) {
        lyd_print_mem(&json_str, root, LYD_JSON, LYP_WITHSIBLINGS | LYP_FORMAT);
        printf("\n%s\n", json_str);
    }

cleanup:
    sr_free_change_iter(it);
    return json_str;
}

char * get_data_json(sr_session_ctx_t *session, const char *module_name) {
    struct lyd_node *data;
    char *json_str = NULL;
    int rc;
    char xpath[100];

    sprintf(xpath, "/%s:ric", module_name);

    rc = sr_get_data(session, xpath, 0, 0, 0, &data);
    if (rc != SR_ERR_OK) {
        fprintf(stdout, "\nsr_get_data failed: %d\n", rc);
        return NULL;
    }

    if (data) {
        lyd_print_mem(&json_str, data, LYD_JSON, LYP_WITHSIBLINGS | LYP_FORMAT);
    }
    return json_str;
}

sr_val_t *get_val(sr_val_t *val, size_t i) {
	return &val[i];
}