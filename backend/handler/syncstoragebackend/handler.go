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

package syncstoragebackend

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/cubefs/cubefs-dashboard/backend/helper/codes"
	"github.com/cubefs/cubefs-dashboard/backend/helper/ginutils"
	syncservice "github.com/cubefs/cubefs-dashboard/backend/service/sync"
)

type idParam struct {
	Id int64 `form:"id" json:"id" binding:"required"`
}

func List(c *gin.Context) {
	data, err := syncservice.ListStorageBackends(c)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}

func Create(c *gin.Context) {
	input := &syncservice.StorageBackendInput{}
	if !ginutils.Check(c, input) {
		return
	}
	data, err := syncservice.CreateStorageBackend(c, input)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}

func Update(c *gin.Context) {
	p := &idParam{}
	if !ginutils.Check(c, p) {
		return
	}
	input := &syncservice.StorageBackendInput{}
	if err := c.ShouldBindJSON(input); err != nil {
		ginutils.Send(c, codes.InvalidArgs.Code(), err.Error(), nil)
		return
	}
	data, err := syncservice.UpdateStorageBackend(c, p.Id, input)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}

func Delete(c *gin.Context) {
	p := &idParam{}
	if !ginutils.Check(c, p) {
		return
	}
	if err := syncservice.DeleteStorageBackend(c, p.Id); err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), nil)
}

// GetConfig returns the resolved S3 config (with plaintext AK/SK) for rule/task JSON preview.
func GetConfig(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id == 0 {
		ginutils.Send(c, codes.InvalidArgs.Code(), "invalid id", nil)
		return
	}
	data, err := syncservice.GetS3BackendConfig(c, id)
	if err != nil {
		syncservice.SendError(c, err)
		return
	}
	ginutils.Send(c, codes.OK.Code(), codes.OK.Msg(), data)
}
