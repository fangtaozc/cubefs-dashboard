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

package synctask

import (
	"io"

	"github.com/gin-gonic/gin"

	"github.com/cubefs/cubefs-dashboard/backend/helper/codes"
	"github.com/cubefs/cubefs-dashboard/backend/helper/ginutils"
	syncservice "github.com/cubefs/cubefs-dashboard/backend/service/sync"
)

type TaskIDInput struct {
	Id string `form:"id" json:"id" binding:"required"`
}

type TaskListInput struct {
	Status string `form:"status"`
	RuleID string `form:"ruleID"`
	Owner  string `form:"owner"`
}

type TaskExportInput struct {
	Since string `form:"since"`
}

func List(c *gin.Context) {
	input := &TaskListInput{}
	if !ginutils.Check(c, input) {
		return
	}
	data, err := syncservice.ListTasks(c, input.Status, input.RuleID, input.Owner)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}

func Get(c *gin.Context) {
	input := &TaskIDInput{}
	if !ginutils.Check(c, input) {
		return
	}
	data, err := syncservice.GetTask(c, input.Id)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}

func Cancel(c *gin.Context) {
	input := &TaskIDInput{}
	if !ginutils.Check(c, input) {
		return
	}
	data, err := syncservice.CancelTask(c, input.Id)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}

func Retry(c *gin.Context) {
	input := &TaskIDInput{}
	if !ginutils.Check(c, input) {
		return
	}
	data, err := syncservice.RetryTask(c, input.Id)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}

func Delete(c *gin.Context) {
	input := &TaskIDInput{}
	if !ginutils.Check(c, input) {
		return
	}
	data, err := syncservice.DeleteTask(c, input.Id)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}

func Export(c *gin.Context) {
	input := &TaskExportInput{}
	if !ginutils.Check(c, input) {
		return
	}
	resp, err := syncservice.ExportTasks(c, input.Since)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	defer resp.Body.Close()

	c.Header("Content-Type", resp.Header.Get("Content-Type"))
	c.Header("Content-Disposition", resp.Header.Get("Content-Disposition"))
	c.Status(resp.StatusCode)
	_, _ = io.Copy(c.Writer, resp.Body)
}
