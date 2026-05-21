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
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/cubefs/cubefs-dashboard/backend/helper/types"
	"github.com/cubefs/cubefs-dashboard/backend/model"
)

type StorageBackendInput struct {
	Name         string `json:"name" binding:"required"`
	Kind         string `json:"kind" binding:"required"`
	Endpoint     string `json:"endpoint" binding:"required"`
	Bucket       string `json:"bucket" binding:"required"`
	Region       string `json:"region"`
	AccessKey    string `json:"access_key" binding:"required"`
	SecretKey    string `json:"secret_key" binding:"required"`
	UsePathStyle bool   `json:"use_path_style"`
	InsecureTLS  bool   `json:"insecure_tls"`
	Remark       string `json:"remark"`
}

type StorageBackendView struct {
	Id              int64  `json:"id"`
	ClusterName     string `json:"cluster_name"`
	Name            string `json:"name"`
	Kind            string `json:"kind"`
	Endpoint        string `json:"endpoint"`
	Bucket          string `json:"bucket"`
	Region          string `json:"region"`
	AccessKeyMasked string `json:"access_key_masked"`
	UsePathStyle    bool   `json:"use_path_style"`
	InsecureTLS     bool   `json:"insecure_tls"`
	Remark          string `json:"remark"`
	CreateTime      string `json:"create_time"`
	UpdateTime      string `json:"update_time"`
}

// S3BackendConfig is the inline credential config injected into rule/task JSON for syncnode.
type S3BackendConfig struct {
	Kind         string `json:"kind"`
	Endpoint     string `json:"endpoint"`
	Bucket       string `json:"bucket"`
	Region       string `json:"region,omitempty"`
	AccessKey    string `json:"accessKey"`
	SecretKey    string `json:"secretKey"`
	UsePathStyle bool   `json:"usePathStyle,omitempty"`
	InsecureTLS  bool   `json:"insecureTLS,omitempty"`
}

func maskKey(key string) string {
	if len(key) <= 6 {
		return "***"
	}
	return key[:3] + "***" + key[len(key)-3:]
}

func backendToView(b *model.SyncStorageBackend, ak string) StorageBackendView {
	return StorageBackendView{
		Id:              b.Id,
		ClusterName:     b.ClusterName,
		Name:            b.Name,
		Kind:            b.Kind,
		Endpoint:        b.Endpoint,
		Bucket:          b.Bucket,
		Region:          b.Region,
		AccessKeyMasked: maskKey(ak),
		UsePathStyle:    b.UsePathStyle,
		InsecureTLS:     b.InsecureTLS,
		Remark:          b.Remark,
		CreateTime:      b.CreateTime.Format("2006-01-02 15:04:05"),
		UpdateTime:      b.UpdateTime.Format("2006-01-02 15:04:05"),
	}
}

func ListStorageBackends(c *gin.Context) (interface{}, error) {
	cluster, err := ResolveCluster(c)
	if err != nil {
		return nil, err
	}
	list, err := model.ListSyncStorageBackends(cluster.Name)
	if err != nil {
		return nil, err
	}
	views := make([]StorageBackendView, 0, len(list))
	for i := range list {
		views = append(views, backendToView(&list[i], string(list[i].AccessKey)))
	}
	return views, nil
}

func CreateStorageBackend(c *gin.Context, input *StorageBackendInput) (interface{}, error) {
	cluster, err := ResolveCluster(c)
	if err != nil {
		return nil, err
	}
	b := &model.SyncStorageBackend{
		ClusterName:  cluster.Name,
		Name:         input.Name,
		Kind:         input.Kind,
		Endpoint:     input.Endpoint,
		Bucket:       input.Bucket,
		Region:       input.Region,
		AccessKey:    types.EncryptStr(input.AccessKey),
		SecretKey:    types.EncryptStr(input.SecretKey),
		UsePathStyle: input.UsePathStyle,
		InsecureTLS:  input.InsecureTLS,
		Remark:       input.Remark,
	}
	if err := b.Create(); err != nil {
		return nil, err
	}
	return backendToView(b, input.AccessKey), nil
}

func UpdateStorageBackend(c *gin.Context, id int64, input *StorageBackendInput) (interface{}, error) {
	if id == 0 {
		return nil, errors.New("id is required")
	}
	cluster, err := ResolveCluster(c)
	if err != nil {
		return nil, err
	}
	b := &model.SyncStorageBackend{}
	if err := b.FindById(id); err != nil {
		return nil, fmt.Errorf("storage backend not found: %w", err)
	}
	if b.ClusterName != cluster.Name {
		return nil, errors.New("storage backend not found in this cluster")
	}
	set := map[string]interface{}{
		"name":           input.Name,
		"kind":           input.Kind,
		"endpoint":       input.Endpoint,
		"bucket":         input.Bucket,
		"region":         input.Region,
		"use_path_style": input.UsePathStyle,
		"insecure_tls":   input.InsecureTLS,
		"remark":         input.Remark,
	}
	if input.AccessKey != "" {
		set["access_key"] = types.EncryptStr(input.AccessKey)
	}
	if input.SecretKey != "" {
		set["secret_key"] = types.EncryptStr(input.SecretKey)
	}
	if err := b.Update(id, set); err != nil {
		return nil, err
	}
	if err := b.FindById(id); err != nil {
		return nil, err
	}
	ak := input.AccessKey
	if ak == "" {
		ak = string(b.AccessKey)
	}
	return backendToView(b, ak), nil
}

func DeleteStorageBackend(c *gin.Context, id int64) error {
	if id == 0 {
		return errors.New("id is required")
	}
	cluster, err := ResolveCluster(c)
	if err != nil {
		return err
	}
	b := &model.SyncStorageBackend{}
	if err := b.FindById(id); err != nil {
		return fmt.Errorf("storage backend not found: %w", err)
	}
	if b.ClusterName != cluster.Name {
		return errors.New("storage backend not found in this cluster")
	}
	return b.Delete(id)
}

// GetS3BackendConfig returns the inline S3 config for injecting into rule/task JSON.
func GetS3BackendConfig(c *gin.Context, id int64) (*S3BackendConfig, error) {
	cluster, err := ResolveCluster(c)
	if err != nil {
		return nil, err
	}
	b := &model.SyncStorageBackend{}
	if err := b.FindById(id); err != nil {
		return nil, fmt.Errorf("storage backend not found: %w", err)
	}
	if b.ClusterName != cluster.Name {
		return nil, errors.New("storage backend not found in this cluster")
	}
	return &S3BackendConfig{
		Kind:         b.Kind,
		Endpoint:     b.Endpoint,
		Bucket:       b.Bucket,
		Region:       b.Region,
		AccessKey:    string(b.AccessKey),
		SecretKey:    string(b.SecretKey),
		UsePathStyle: b.UsePathStyle,
		InsecureTLS:  b.InsecureTLS,
	}, nil
}
