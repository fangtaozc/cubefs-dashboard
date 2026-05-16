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

package sync

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/cubefs/cubefs-dashboard/backend/helper"
	"github.com/cubefs/cubefs-dashboard/backend/helper/codes"
	"github.com/cubefs/cubefs-dashboard/backend/helper/ginutils"
	"github.com/cubefs/cubefs-dashboard/backend/helper/httputils"
	"github.com/cubefs/cubefs-dashboard/backend/model"
	clusterService "github.com/cubefs/cubefs-dashboard/backend/service/cluster"
)

const (
	DefaultNodeHTTPPort = 17911

	upstreamMaster = "master"
	upstreamNode   = "syncnode"
)

type upstreamReply struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

type UpstreamError struct {
	DashboardCode int
	Msg           string
	Upstream      string
	SyncCode      int
}

func (e *UpstreamError) Error() string {
	return e.Msg
}

func (e *UpstreamError) Payload() map[string]interface{} {
	return map[string]interface{}{
		"upstream":  e.Upstream,
		"sync_code": e.SyncCode,
	}
}

func SendError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	var upstreamErr *UpstreamError
	if errors.As(err, &upstreamErr) {
		ginutils.Send(c, upstreamErr.DashboardCode, upstreamErr.Msg, upstreamErr.Payload())
		return
	}
	ginutils.Send(c, codes.ThirdPartyError.Code(), err.Error(), nil)
}

func ResolveCluster(c *gin.Context) (*model.Cluster, error) {
	name := c.Param(ginutils.Cluster)
	cluster, err := new(model.Cluster).FindName(name)
	if err != nil {
		return nil, err
	}
	if cluster == nil || len(cluster.MasterAddr) == 0 {
		return nil, fmt.Errorf("cluster(%s) master addr is empty", name)
	}
	return cluster, nil
}

func ResolveMaster(c *gin.Context) (*model.Cluster, string, error) {
	cluster, err := ResolveCluster(c)
	if err != nil {
		return nil, "", err
	}
	addr := cluster.MasterAddr[0]
	view, err := clusterService.Get(c, addr)
	if err != nil || view == nil || view.LeaderAddr == "" {
		return cluster, addr, nil
	}
	return cluster, view.LeaderAddr, nil
}

func ResolveNodeAdminAddr(cluster *model.Cluster, syncNodeAddr string) (string, error) {
	if cluster == nil {
		return "", errors.New("cluster is nil")
	}
	host := helper.GetIp(syncNodeAddr)
	if host == "" {
		return "", errors.New("sync node addr is empty")
	}
	port := cluster.SyncNodeHTTPPort
	if port <= 0 {
		port = DefaultNodeHTTPPort
	}
	return fmt.Sprintf("http://%s:%d", host, port), nil
}

func ListRules(c *gin.Context, state string) (interface{}, error) {
	return doMasterJSON(c, http.MethodGet, "/syncRule/list", buildValues(map[string]string{
		"state": state,
	}), nil)
}

func GetRule(c *gin.Context, id string) (interface{}, error) {
	return doMasterJSON(c, http.MethodGet, "/syncRule/get", buildValues(map[string]string{
		"id": id,
	}), nil)
}

func CreateRule(c *gin.Context, body interface{}) (interface{}, error) {
	return doMasterJSON(c, http.MethodPost, "/syncRule/create", nil, body)
}

func UpdateRule(c *gin.Context, body interface{}) (interface{}, error) {
	return doMasterJSON(c, http.MethodPost, "/syncRule/update", nil, body)
}

func DeleteRule(c *gin.Context, id string) (interface{}, error) {
	return doMasterJSON(c, http.MethodPost, "/syncRule/delete", buildValues(map[string]string{
		"id": id,
	}), nil)
}

func PauseRule(c *gin.Context, id string) (interface{}, error) {
	return doMasterJSON(c, http.MethodPost, "/syncRule/pause", buildValues(map[string]string{
		"id": id,
	}), nil)
}

func ResumeRule(c *gin.Context, id string) (interface{}, error) {
	return doMasterJSON(c, http.MethodPost, "/syncRule/resume", buildValues(map[string]string{
		"id": id,
	}), nil)
}

func TriggerRule(c *gin.Context, id string) (interface{}, error) {
	return doMasterJSON(c, http.MethodPost, "/syncRule/trigger", buildValues(map[string]string{
		"id": id,
	}), nil)
}

func ListTasks(c *gin.Context, status, ruleID, owner string) (interface{}, error) {
	return doMasterJSON(c, http.MethodGet, "/syncTask/list", buildValues(map[string]string{
		"status": status,
		"ruleID": ruleID,
		"owner":  owner,
	}), nil)
}

func GetTask(c *gin.Context, id string) (interface{}, error) {
	return doMasterJSON(c, http.MethodGet, "/syncTask/get", buildValues(map[string]string{
		"id": id,
	}), nil)
}

func CancelTask(c *gin.Context, id string) (interface{}, error) {
	return doMasterJSON(c, http.MethodPost, "/syncTask/cancel", buildValues(map[string]string{
		"id": id,
	}), nil)
}

func RetryTask(c *gin.Context, id string) (interface{}, error) {
	return doMasterJSON(c, http.MethodPost, "/syncTask/retry", buildValues(map[string]string{
		"id": id,
	}), nil)
}

func DeleteTask(c *gin.Context, id string) (interface{}, error) {
	return doMasterJSON(c, http.MethodPost, "/syncTask/delete", buildValues(map[string]string{
		"id": id,
	}), nil)
}

func ExportTasks(c *gin.Context, since string) (*http.Response, error) {
	cluster, leaderAddr, err := ResolveMaster(c)
	if err != nil {
		return nil, err
	}
	return doRawRequest(c, upstreamMaster, http.MethodGet, "http://"+leaderAddr, "/syncTask/export", buildValues(map[string]string{
		"since": since,
	}), nil, string(cluster.SyncAdminToken))
}

func ListNodes(c *gin.Context) (interface{}, error) {
	return doMasterJSON(c, http.MethodGet, "/syncNode/list", nil, nil)
}

func DispatchTask(c *gin.Context, body interface{}) (interface{}, error) {
	return doMasterJSON(c, http.MethodPost, "/syncNode/dispatch", nil, body)
}

func ListNodeTasks(c *gin.Context, addr, status string) (interface{}, error) {
	return doMasterJSON(c, http.MethodGet, "/syncNode/tasks", buildValues(map[string]string{
		"addr":   addr,
		"status": status,
	}), nil)
}

func DrainNode(c *gin.Context, addr string) (interface{}, error) {
	return doMasterJSON(c, http.MethodPost, "/syncNode/drain", buildValues(map[string]string{
		"addr": addr,
	}), nil)
}

func RestoreNode(c *gin.Context, addr string) (interface{}, error) {
	return doMasterJSON(c, http.MethodPost, "/syncNode/restore", buildValues(map[string]string{
		"addr": addr,
	}), nil)
}

func DecommissionNode(c *gin.Context, addr string, force bool) (interface{}, error) {
	values := buildValues(map[string]string{
		"addr": addr,
	})
	values.Set("force", fmt.Sprintf("%t", force))
	return doMasterJSON(c, http.MethodPost, "/syncNode/decommission", values, nil)
}

func GetNodeVersion(c *gin.Context, syncNodeAddr string) (interface{}, error) {
	return doNodeJSON(c, http.MethodGet, syncNodeAddr, "/admin/syncnode/version", nil)
}

func GetNodeStat(c *gin.Context, syncNodeAddr string) (interface{}, error) {
	return doNodeJSON(c, http.MethodGet, syncNodeAddr, "/admin/syncnode/stat", nil)
}

func ReloadNode(c *gin.Context, syncNodeAddr string) (interface{}, error) {
	return doNodeJSON(c, http.MethodPost, syncNodeAddr, "/admin/syncnode/reload", nil)
}

func doMasterJSON(c *gin.Context, method, path string, values url.Values, body interface{}) (interface{}, error) {
	cluster, leaderAddr, err := ResolveMaster(c)
	if err != nil {
		return nil, err
	}
	return doJSONRequest(c, upstreamMaster, method, "http://"+leaderAddr, path, values, body, string(cluster.SyncAdminToken))
}

func doNodeJSON(c *gin.Context, method, syncNodeAddr, path string, body interface{}) (interface{}, error) {
	cluster, err := ResolveCluster(c)
	if err != nil {
		return nil, err
	}
	adminAddr, err := ResolveNodeAdminAddr(cluster, syncNodeAddr)
	if err != nil {
		return nil, err
	}
	return doJSONRequest(c, upstreamNode, method, adminAddr, path, nil, body, string(cluster.SyncAdminToken))
}

func doJSONRequest(c *gin.Context, upstream, method, baseURL, path string, values url.Values, body interface{}, token string) (interface{}, error) {
	resp, err := doRawRequest(c, upstream, method, baseURL, path, values, body, token)
	if err != nil {
		return nil, err
	}
	return parseJSONResponse(upstream, resp)
}

func doRawRequest(c *gin.Context, upstream, method, baseURL, path string, values url.Values, body interface{}, token string) (*http.Response, error) {
	bodyReader, headers, err := buildRequestBody(body, token)
	if err != nil {
		return nil, err
	}
	reqURL := baseURL + path
	if len(values) > 0 {
		reqURL = reqURL + "?" + values.Encode()
	}
	resp, err := httputils.DoRequestNoCookie(c, reqURL, method, bodyReader, headers)
	if err != nil {
		return nil, &UpstreamError{
			DashboardCode: codes.ThirdPartyError.Code(),
			Msg:           err.Error(),
			Upstream:      upstream,
			SyncCode:      0,
		}
	}
	return resp, nil
}

func buildRequestBody(body interface{}, token string) (io.Reader, map[string]string, error) {
	headers := make(map[string]string)
	if token != "" {
		headers["X-Sync-Token"] = token
	}
	if body == nil {
		return nil, headers, nil
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, nil, err
	}
	headers["Content-Type"] = "application/json"
	return bytes.NewReader(data), headers, nil
}

func parseJSONResponse(upstream string, resp *http.Response) (interface{}, error) {
	if resp == nil {
		return nil, &UpstreamError{
			DashboardCode: codes.ThirdPartyError.Code(),
			Msg:           "empty upstream response",
			Upstream:      upstream,
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &UpstreamError{
			DashboardCode: codes.ThirdPartyError.Code(),
			Msg:           err.Error(),
			Upstream:      upstream,
		}
	}

	reply := upstreamReply{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &reply); err != nil {
			msg := strings.TrimSpace(string(body))
			if msg == "" {
				msg = err.Error()
			}
			return nil, &UpstreamError{
				DashboardCode: mapUpstreamError(upstream, resp.StatusCode, 0, msg),
				Msg:           msg,
				Upstream:      upstream,
			}
		}
	}

	if resp.StatusCode == http.StatusOK && reply.Code == 0 {
		if len(reply.Data) == 0 || string(reply.Data) == "null" {
			return nil, nil
		}
		var data interface{}
		if err := json.Unmarshal(reply.Data, &data); err != nil {
			return nil, &UpstreamError{
				DashboardCode: codes.ThirdPartyError.Code(),
				Msg:           err.Error(),
				Upstream:      upstream,
			}
		}
		return data, nil
	}

	msg := reply.Msg
	if msg == "" {
		msg = strings.TrimSpace(string(body))
	}
	if msg == "" {
		msg = resp.Status
	}

	return nil, &UpstreamError{
		DashboardCode: mapUpstreamError(upstream, resp.StatusCode, reply.Code, msg),
		Msg:           msg,
		Upstream:      upstream,
		SyncCode:      reply.Code,
	}
}

func mapUpstreamError(upstream string, statusCode, syncCode int, msg string) int {
	switch statusCode {
	case http.StatusUnauthorized:
		return codes.Unauthorized.Code()
	case http.StatusForbidden:
		return codes.Forbidden.Code()
	case http.StatusNotFound:
		return codes.NotFound.Code()
	}

	switch syncCode {
	case 2:
		return codes.InvalidArgs.Code()
	case 4:
		return http.StatusServiceUnavailable
	case 1014, 1015, 1016:
		return codes.Conflict.Code()
	}

	if syncCode == 1 && strings.Contains(strings.ToLower(msg), "not found") {
		return codes.NotFound.Code()
	}
	if statusCode == http.StatusServiceUnavailable {
		return http.StatusServiceUnavailable
	}
	if upstream == upstreamNode && statusCode >= http.StatusInternalServerError {
		return codes.ThirdPartyError.Code()
	}
	return codes.ThirdPartyError.Code()
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
