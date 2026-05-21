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

package model

import (
	"errors"
	"time"

	"github.com/cubefs/cubefs-dashboard/backend/helper/types"
	"github.com/cubefs/cubefs-dashboard/backend/model/mysql"
)

// SyncStorageBackend stores pre-configured S3-compatible bucket credentials.
// kind: tos / bos / s3
type SyncStorageBackend struct {
	Id           int64            `gorm:"primaryKey;auto_increment" json:"id"`
	ClusterName  string           `gorm:"column:cluster_name;type:varchar(100);not null;default:'';index" json:"cluster_name"`
	Name         string           `gorm:"type:varchar(100);not null;default:''" json:"name"`
	Kind         string           `gorm:"type:varchar(20);not null;default:'s3'" json:"kind"`
	Endpoint     string           `gorm:"type:varchar(500);not null;default:''" json:"endpoint"`
	Bucket       string           `gorm:"type:varchar(255);not null;default:''" json:"bucket"`
	Region       string           `gorm:"type:varchar(100);not null;default:''" json:"region"`
	AccessKey    types.EncryptStr `gorm:"column:access_key;type:varchar(500);not null;default:''" json:"-"`
	SecretKey    types.EncryptStr `gorm:"column:secret_key;type:varchar(1000);not null;default:''" json:"-"`
	UsePathStyle bool             `gorm:"column:use_path_style;type:tinyint(1);not null;default:0" json:"use_path_style"`
	InsecureTLS  bool             `gorm:"column:insecure_tls;type:tinyint(1);not null;default:0" json:"insecure_tls"`
	Remark       string           `gorm:"type:varchar(500);not null;default:''" json:"remark"`
	CreateTime   time.Time        `gorm:"column:create_time" json:"create_time"`
	UpdateTime   time.Time        `gorm:"column:update_time" json:"update_time"`
}

func (s *SyncStorageBackend) TableName() string {
	return "sync_storage_backend"
}

func (s *SyncStorageBackend) Create() error {
	now := time.Now()
	s.CreateTime = now
	s.UpdateTime = now
	return mysql.GetDB().Create(s).Error
}

func (s *SyncStorageBackend) Update(id int64, set map[string]interface{}) error {
	if id == 0 {
		return errors.New("id is required")
	}
	if len(set) == 0 {
		return nil
	}
	set["update_time"] = time.Now()
	return mysql.GetDB().Model(&SyncStorageBackend{}).Where("id = ?", id).Updates(set).Error
}

func (s *SyncStorageBackend) Delete(id int64) error {
	if id == 0 {
		return errors.New("id is required")
	}
	return mysql.GetDB().Where("id = ?", id).Delete(&SyncStorageBackend{}).Error
}

func (s *SyncStorageBackend) FindById(id int64) error {
	return mysql.GetDB().Where("id = ?", id).First(s).Error
}

func ListSyncStorageBackends(clusterName string) ([]SyncStorageBackend, error) {
	var list []SyncStorageBackend
	query := mysql.GetDB().Model(&SyncStorageBackend{})
	if clusterName != "" {
		query = query.Where("cluster_name = ?", clusterName)
	}
	err := query.Order("id asc").Find(&list).Error
	return list, err
}
