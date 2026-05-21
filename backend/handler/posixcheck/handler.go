// Copyright 2026 The CubeFS Authors.
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

// Package posixcheck implements the HTTP handlers for the POSIX compliance
// (pjd-fstest) feature. All routes are exposed under
// /api/cubefs/console/cfs/:cluster/posixCheck/* and gated by the dashboard's
// standard auth middleware.
package posixcheck

import (
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/cubefs/cubefs-dashboard/backend/helper/codes"
	"github.com/cubefs/cubefs-dashboard/backend/helper/ginutils"
	"github.com/cubefs/cubefs-dashboard/backend/model"
	pcsvc "github.com/cubefs/cubefs-dashboard/backend/service/posixcheck"
)

// Singleton Runner — built lazily on first use so the dashboard can start
// outside a Kubernetes pod (in which case the feature simply reports
// "not running in-cluster" to the caller).
var (
	runnerOnce sync.Once
	runnerInst *pcsvc.Runner
	runnerErr  error
)

func getRunner() (*pcsvc.Runner, error) {
	runnerOnce.Do(func() {
		runnerInst, runnerErr = pcsvc.NewRunner("")
	})
	return runnerInst, runnerErr
}

func triggerUserName(c *gin.Context) string {
	if u, err := ginutils.GetLoginUser(c); err == nil && u != nil {
		return u.UserName
	}
	return ""
}

// RunInput is the body of POST /posixCheck/run.
type RunInput struct {
	TargetVol   string `json:"target_vol" binding:"required"`
	MountSubdir string `json:"mount_subdir"`
	SuiteImage  string `json:"suite_image"`
	TestFilter  string `json:"test_filter"`
}

// ListInput is the query for GET /posixCheck/list.
type ListInput struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

// IDInput is shared by /get and /cancel.
type IDInput struct {
	Id int64 `form:"id" json:"id" binding:"required"`
}

// Run handles POST /posixCheck/run — creates a new compliance run and
// kicks off the K8s Job asynchronously.
func Run(c *gin.Context) {
	clusterName := c.Param("cluster")
	if clusterName == "" {
		ginutils.Send(c, codes.InvalidArgs.Code(), "missing cluster path param", nil)
		return
	}
	input := &RunInput{}
	if !ginutils.Check(c, input) {
		return
	}
	r, err := getRunner()
	if err != nil {
		ginutils.Send(c, codes.ResultError.Code(), "posix-check runner unavailable: "+err.Error(), nil)
		return
	}
	run, err := r.Submit(pcsvc.SubmitParams{
		ClusterName: clusterName,
		TargetVol:   input.TargetVol,
		MountSubdir: input.MountSubdir,
		SuiteImage:  input.SuiteImage,
		TestFilter:  input.TestFilter,
		TriggerUser: triggerUserName(c),
	})
	if err != nil {
		ginutils.Send(c, codes.ResultError.Code(), err.Error(), run)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), run)
}

// List handles GET /posixCheck/list — returns recent runs for the cluster.
func List(c *gin.Context) {
	clusterName := c.Param("cluster")
	input := &ListInput{}
	if !ginutils.Check(c, input) {
		return
	}
	rows, total, err := model.ListPosixCheckRuns(model.ListPosixCheckRunsParam{
		ClusterName: clusterName,
		Page:        input.Page,
		PageSize:    input.PageSize,
	})
	if err != nil {
		ginutils.Send(c, codes.ResultError.Code(), err.Error(), nil)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), gin.H{
		"data":  rows,
		"total": total,
	})
}

// Get handles GET /posixCheck/get?id=N — full run details + failure list.
func Get(c *gin.Context) {
	input := &IDInput{}
	if !ginutils.Check(c, input) {
		return
	}
	run, err := model.GetPosixCheckRun(input.Id)
	if err != nil {
		ginutils.Send(c, codes.ResultError.Code(), err.Error(), nil)
		return
	}
	failures, err := model.ListPosixCheckFailures(input.Id)
	if err != nil {
		ginutils.Send(c, codes.ResultError.Code(), err.Error(), nil)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), gin.H{
		"run":      run,
		"failures": failures,
	})
}

// Cancel handles POST /posixCheck/cancel?id=N.
func Cancel(c *gin.Context) {
	input := &IDInput{}
	if !ginutils.Check(c, input) {
		return
	}
	r, err := getRunner()
	if err != nil {
		ginutils.Send(c, codes.ResultError.Code(), err.Error(), nil)
		return
	}
	if err := r.Cancel(input.Id); err != nil {
		ginutils.Send(c, codes.ResultError.Code(), err.Error(), nil)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), gin.H{"id": input.Id, "status": "cancelled"})
}

// Delete handles POST /posixCheck/delete?id=N. Removes the run record and
// its failure rows; for in-flight runs the K8s Job is killed first.
func Delete(c *gin.Context) {
	input := &IDInput{}
	if !ginutils.Check(c, input) {
		return
	}
	r, err := getRunner()
	if err != nil {
		ginutils.Send(c, codes.ResultError.Code(), err.Error(), nil)
		return
	}
	if err := r.Delete(input.Id); err != nil {
		ginutils.Send(c, codes.ResultError.Code(), err.Error(), nil)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), gin.H{"id": input.Id, "status": "deleted"})
}
