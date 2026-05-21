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

package benchrule

import (
	"github.com/gin-gonic/gin"

	"github.com/cubefs/cubefs-dashboard/backend/helper/codes"
	"github.com/cubefs/cubefs-dashboard/backend/helper/ginutils"
	benchservice "github.com/cubefs/cubefs-dashboard/backend/service/bench"
	syncservice "github.com/cubefs/cubefs-dashboard/backend/service/sync"
)

type RuleIDInput struct {
	Id string `form:"id" json:"id" binding:"required"`
}

type RuleListInput struct {
	Status string `form:"status"`
}

func List(c *gin.Context) {
	input := &RuleListInput{}
	if !ginutils.Check(c, input) {
		return
	}
	data, err := benchservice.ListBenchRules(c, input.Status)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}

func Get(c *gin.Context) {
	input := &RuleIDInput{}
	if !ginutils.Check(c, input) {
		return
	}
	data, err := benchservice.GetBenchRule(c, input.Id)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}

func Create(c *gin.Context) {
	input, ok := parseRuleConfig(c)
	if !ok {
		return
	}
	data, err := benchservice.CreateBenchRule(c, input)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}

func Update(c *gin.Context) {
	input, ok := parseRuleConfig(c)
	if !ok {
		return
	}
	data, err := benchservice.UpdateBenchRule(c, input)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}

func Delete(c *gin.Context) {
	input := &RuleIDInput{}
	if !ginutils.Check(c, input) {
		return
	}
	data, err := benchservice.DeleteBenchRule(c, input.Id)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}

func Trigger(c *gin.Context) {
	input := &RuleIDInput{}
	if !ginutils.Check(c, input) {
		return
	}
	data, err := benchservice.TriggerBenchRule(c, input.Id)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}

func parseRuleConfig(c *gin.Context) (map[string]interface{}, bool) {
	input := make(map[string]interface{})
	if err := c.ShouldBindJSON(&input); err != nil {
		ginutils.Send(c, codes.InvalidArgs.Code(), err.Error(), nil)
		return nil, false
	}
	if len(input) == 0 {
		ginutils.Send(c, codes.InvalidArgs.Code(), "bench rule config is empty", nil)
		return nil, false
	}
	return input, true
}
