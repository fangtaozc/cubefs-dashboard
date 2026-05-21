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

package model

import (
	"time"

	"github.com/cubefs/cubefs-dashboard/backend/model/mysql"
	"gorm.io/gorm"
)

// PosixCheckRunStatus is the lifecycle state of one POSIX compliance run.
type PosixCheckRunStatus string

const (
	PosixCheckRunPending   PosixCheckRunStatus = "pending"
	PosixCheckRunRunning   PosixCheckRunStatus = "running"
	PosixCheckRunDone      PosixCheckRunStatus = "done"
	PosixCheckRunFailed    PosixCheckRunStatus = "failed"
	PosixCheckRunCancelled PosixCheckRunStatus = "cancelled"
)

// PosixCheckRun is one execution of the pjd-fstest suite against a target
// volume. It is the parent record; per-test failure details land in
// PosixCheckFailure rows linked by RunID.
type PosixCheckRun struct {
	Id          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ClusterName string    `gorm:"type:varchar(128);not null;default:''" json:"cluster_name"`
	TargetVol   string    `gorm:"type:varchar(128);not null;default:''" json:"target_vol"`
	MountSubdir string    `gorm:"type:varchar(256);not null;default:''" json:"mount_subdir"`
	SuiteImage  string    `gorm:"type:varchar(256);not null;default:''" json:"suite_image"`
	TestFilter  string    `gorm:"type:varchar(512);not null;default:''" json:"test_filter"`
	K8sJobName  string    `gorm:"type:varchar(128);not null;default:''" json:"k8s_job_name"`
	Status      string    `gorm:"type:varchar(32);not null;default:'pending';index" json:"status"`
	PassCount   int       `gorm:"not null;default:0" json:"pass_count"`
	FailCount   int       `gorm:"not null;default:0" json:"fail_count"`
	SkipCount   int       `gorm:"not null;default:0" json:"skip_count"`
	TotalCount  int       `gorm:"not null;default:0" json:"total_count"`
	DurationSec int       `gorm:"not null;default:0" json:"duration_sec"`
	TriggerUser string    `gorm:"type:varchar(64);not null;default:''" json:"trigger_user"`
	ErrorMsg    string    `gorm:"type:varchar(1024);not null;default:''" json:"error_msg,omitempty"`
	CreatedAt   time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
	FinishedAt  time.Time `gorm:"default:null" json:"finished_at,omitempty"`
}

func (PosixCheckRun) TableName() string { return "posix_check_runs" }

// PosixCheckFailure is one failed test case from a run. Successful tests are
// only counted (PosixCheckRun.PassCount); only failures are detailed here so
// the table stays small even for runs with thousands of cases.
type PosixCheckFailure struct {
	Id          int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	RunId       int64  `gorm:"not null;index" json:"run_id"`
	TestFile    string `gorm:"type:varchar(256);not null;default:''" json:"test_file"`
	TestNumber  int    `gorm:"not null;default:0" json:"test_number"`
	Description string `gorm:"type:varchar(512);not null;default:''" json:"description"`
	Syscall     string `gorm:"type:varchar(64);not null;default:''" json:"syscall"`
	Expected    string `gorm:"type:varchar(512);not null;default:''" json:"expected"`
	Actual      string `gorm:"type:varchar(512);not null;default:''" json:"actual"`
}

func (PosixCheckFailure) TableName() string { return "posix_check_failures" }

// CreatePosixCheckRun inserts a new pending run record and returns its id.
func CreatePosixCheckRun(r *PosixCheckRun) error {
	return mysql.GetDB().Create(r).Error
}

// UpdatePosixCheckRun applies the supplied non-zero fields to the row with
// the given id. Caller is responsible for setting the relevant subset.
func UpdatePosixCheckRun(id int64, set map[string]interface{}) error {
	return mysql.GetDB().Model(&PosixCheckRun{}).Where("id = ?", id).Updates(set).Error
}

// GetPosixCheckRun fetches one run by id.
func GetPosixCheckRun(id int64) (*PosixCheckRun, error) {
	r := new(PosixCheckRun)
	if err := mysql.GetDB().Where("id = ?", id).First(r).Error; err != nil {
		return nil, err
	}
	return r, nil
}

// ListPosixCheckRuns returns runs filtered by cluster (empty = all),
// ordered by created_at desc.
type ListPosixCheckRunsParam struct {
	ClusterName string
	Page        int
	PageSize    int
}

func ListPosixCheckRuns(p ListPosixCheckRunsParam) ([]PosixCheckRun, int64, error) {
	db := mysql.GetDB().Model(&PosixCheckRun{})
	if p.ClusterName != "" {
		db = db.Where("cluster_name = ?", p.ClusterName)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if p.PageSize <= 0 {
		p.PageSize = 20
	}
	if p.Page <= 0 {
		p.Page = 1
	}
	rows := make([]PosixCheckRun, 0)
	err := db.Order("created_at desc").Offset((p.Page - 1) * p.PageSize).Limit(p.PageSize).Find(&rows).Error
	return rows, total, err
}

// BatchInsertPosixCheckFailures records all failures of a single run in one
// query. Empty input is a no-op.
func BatchInsertPosixCheckFailures(failures []PosixCheckFailure) error {
	if len(failures) == 0 {
		return nil
	}
	return mysql.GetDB().Create(&failures).Error
}

// ListPosixCheckFailures returns all failure rows for a run, ordered by
// test file path then test number.
func ListPosixCheckFailures(runId int64) ([]PosixCheckFailure, error) {
	rows := make([]PosixCheckFailure, 0)
	err := mysql.GetDB().Where("run_id = ?", runId).
		Order("test_file asc, test_number asc").
		Find(&rows).Error
	return rows, err
}

// DeletePosixCheckRun removes a run record and all its associated failure
// rows in a single transaction. Caller should already have stopped the
// underlying K8s Job (if any) before calling this.
func DeletePosixCheckRun(id int64) error {
	return mysql.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("run_id = ?", id).Delete(&PosixCheckFailure{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).Delete(&PosixCheckRun{}).Error
	})
}

// MigratePosixCheck applies AutoMigrate for both tables. Called from the
// initial gormigrate migration so the rest of the codebase can rely on
// the tables existing.
func MigratePosixCheck(tx *gorm.DB) error {
	if err := tx.AutoMigrate(&PosixCheckRun{}); err != nil {
		return err
	}
	return tx.AutoMigrate(&PosixCheckFailure{})
}
