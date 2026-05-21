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

package benchtask

import (
	"github.com/gin-gonic/gin"

	"github.com/cubefs/cubefs-dashboard/backend/helper/codes"
	"github.com/cubefs/cubefs-dashboard/backend/helper/ginutils"
	benchservice "github.com/cubefs/cubefs-dashboard/backend/service/bench"
	syncservice "github.com/cubefs/cubefs-dashboard/backend/service/sync"
)

type TaskIDInput struct {
	Id string `form:"id" json:"id" binding:"required"`
}

type TaskListInput struct {
	Status string `form:"status"`
	RuleID string `form:"ruleID"`
}

func List(c *gin.Context) {
	input := &TaskListInput{}
	if !ginutils.Check(c, input) {
		return
	}
	data, err := benchservice.ListBenchTasks(c, input.Status, input.RuleID)
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
	data, err := benchservice.GetBenchTask(c, input.Id)
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
	data, err := benchservice.CancelBenchTask(c, input.Id)
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
	data, err := benchservice.RetryBenchTask(c, input.Id)
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
	data, err := benchservice.DeleteBenchTask(c, input.Id)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}
