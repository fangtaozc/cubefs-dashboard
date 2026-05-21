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

package syncnode

import (
	"github.com/gin-gonic/gin"

	"github.com/cubefs/cubefs-dashboard/backend/helper/codes"
	"github.com/cubefs/cubefs-dashboard/backend/helper/ginutils"
	syncservice "github.com/cubefs/cubefs-dashboard/backend/service/sync"
)

type NodeAddrInput struct {
	Addr string `form:"addr" json:"addr" binding:"required"`
}

type NodeTaskListInput struct {
	Addr   string `form:"addr" binding:"required"`
	Status string `form:"status"`
}

type DecommissionInput struct {
	Addr  string `json:"addr" binding:"required"`
	Force bool   `json:"force"`
}

func List(c *gin.Context) {
	data, err := syncservice.ListNodes(c)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}

func Dispatch(c *gin.Context) {
	input, ok := parseDispatchTask(c)
	if !ok {
		return
	}
	data, err := syncservice.DispatchTask(c, input)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}

func Tasks(c *gin.Context) {
	input := &NodeTaskListInput{}
	if !ginutils.Check(c, input) {
		return
	}
	data, err := syncservice.ListNodeTasks(c, input.Addr, input.Status)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}

func Version(c *gin.Context) {
	input := &NodeAddrInput{}
	if !ginutils.Check(c, input) {
		return
	}
	data, err := syncservice.GetNodeVersion(c, input.Addr)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}

func Stat(c *gin.Context) {
	input := &NodeAddrInput{}
	if !ginutils.Check(c, input) {
		return
	}
	data, err := syncservice.GetNodeStat(c, input.Addr)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}

func Reload(c *gin.Context) {
	input := &NodeAddrInput{}
	if !ginutils.Check(c, input) {
		return
	}
	data, err := syncservice.ReloadNode(c, input.Addr)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}

func Drain(c *gin.Context) {
	input := &NodeAddrInput{}
	if !ginutils.Check(c, input) {
		return
	}
	data, err := syncservice.DrainNode(c, input.Addr)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}

func Restore(c *gin.Context) {
	input := &NodeAddrInput{}
	if !ginutils.Check(c, input) {
		return
	}
	data, err := syncservice.RestoreNode(c, input.Addr)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}

func Decommission(c *gin.Context) {
	input := &DecommissionInput{}
	if !ginutils.Check(c, input) {
		return
	}
	data, err := syncservice.DecommissionNode(c, input.Addr, input.Force)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}

func parseDispatchTask(c *gin.Context) (map[string]interface{}, bool) {
	input := make(map[string]interface{})
	if err := c.ShouldBindJSON(&input); err != nil {
		ginutils.Send(c, codes.InvalidArgs.Code(), err.Error(), nil)
		return nil, false
	}
	delete(input, "cluster_name")
	if len(input) == 0 {
		ginutils.Send(c, codes.InvalidArgs.Code(), "sync task payload is empty", nil)
		return nil, false
	}
	return input, true
}
