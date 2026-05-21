// Copyright 2023 The CubeFS Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package bench

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	syncservice "github.com/cubefs/cubefs-dashboard/backend/service/sync"
)

func ListBenchRules(c *gin.Context, status string) (interface{}, error) {
	return doMasterJSON(c, http.MethodGet, "/benchRule/list", buildValues(map[string]string{
		"status": status,
	}), nil)
}

func GetBenchRule(c *gin.Context, id string) (interface{}, error) {
	return doMasterJSON(c, http.MethodGet, "/benchRule/get", buildValues(map[string]string{
		"id": id,
	}), nil)
}

func CreateBenchRule(c *gin.Context, body interface{}) (interface{}, error) {
	return doMasterJSON(c, http.MethodPost, "/benchRule/create", nil, body)
}

func UpdateBenchRule(c *gin.Context, body interface{}) (interface{}, error) {
	return doMasterJSON(c, http.MethodPost, "/benchRule/update", nil, body)
}

func DeleteBenchRule(c *gin.Context, id string) (interface{}, error) {
	return doMasterJSON(c, http.MethodPost, "/benchRule/delete", buildValues(map[string]string{
		"id": id,
	}), nil)
}

func TriggerBenchRule(c *gin.Context, id string) (interface{}, error) {
	// Fetch the bench rule so we can extract its backendID and inject resolved
	// S3 credentials before forwarding the trigger to the master. The master
	// itself has no access to the credential store (MySQL); the dashboard is
	// the only layer that can resolve BackendID → plaintext AccessKey/SecretKey.
	ruleData, err := doMasterJSON(c, http.MethodGet, "/benchRule/get", buildValues(map[string]string{
		"id": id,
	}), nil)
	if err != nil {
		return nil, err
	}

	var body interface{}
	if ruleMap, ok := ruleData.(map[string]interface{}); ok {
		backendIDStr, _ := ruleMap["backendID"].(string)
		if backendIDStr != "" {
			backendID, perr := strconv.ParseInt(backendIDStr, 10, 64)
			if perr != nil {
				return nil, fmt.Errorf("bench rule %q has invalid backendID %q: %w", id, backendIDStr, perr)
			}
			s3cfg, cerr := syncservice.GetS3BackendConfig(c, backendID)
			if cerr != nil {
				return nil, fmt.Errorf("resolve backend credentials for bench rule %q: %w", id, cerr)
			}
			body = map[string]interface{}{
				"backendEndpoint": map[string]interface{}{
					"kind":            s3cfg.Kind,
					"endpoint":        s3cfg.Endpoint,
					"bucket":          s3cfg.Bucket,
					"region":          s3cfg.Region,
					"accessKey":       s3cfg.AccessKey,
					"secretKey":       s3cfg.SecretKey,
					"usePathStyle":    s3cfg.UsePathStyle,
					"insecureSkipTLS": s3cfg.InsecureTLS,
				},
			}
		}
	}

	return doMasterJSON(c, http.MethodPost, "/benchRule/trigger", buildValues(map[string]string{
		"id": id,
	}), body)
}

func ListBenchTasks(c *gin.Context, status, ruleID string) (interface{}, error) {
	return doMasterJSON(c, http.MethodGet, "/benchTask/list", buildValues(map[string]string{
		"status": status,
		"ruleID": ruleID,
	}), nil)
}

func GetBenchTask(c *gin.Context, id string) (interface{}, error) {
	return doMasterJSON(c, http.MethodGet, "/benchTask/get", buildValues(map[string]string{
		"id": id,
	}), nil)
}

func CancelBenchTask(c *gin.Context, id string) (interface{}, error) {
	return doMasterJSON(c, http.MethodPost, "/benchTask/cancel", buildValues(map[string]string{
		"id": id,
	}), nil)
}

func RetryBenchTask(c *gin.Context, id string) (interface{}, error) {
	return doMasterJSON(c, http.MethodPost, "/benchTask/retry", buildValues(map[string]string{
		"id": id,
	}), nil)
}

func DeleteBenchTask(c *gin.Context, id string) (interface{}, error) {
	return doMasterJSON(c, http.MethodPost, "/benchTask/delete", buildValues(map[string]string{
		"id": id,
	}), nil)
}

func doMasterJSON(c *gin.Context, method, path string, values url.Values, body interface{}) (interface{}, error) {
	cluster, leaderAddr, err := syncservice.ResolveMaster(c)
	if err != nil {
		return nil, err
	}
	return syncservice.DoMasterJSONRequest(c, method, "http://"+leaderAddr, path, values, body, string(cluster.SyncAdminToken))
}

func buildValues(filters map[string]string) url.Values {
	values := url.Values{}
	for key, value := range filters {
		if strings.TrimSpace(value) == "" {
			continue
		}
		values.Set(key, value)
	}
	return values
}
