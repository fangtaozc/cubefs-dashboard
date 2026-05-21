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

package posixcheck

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cubefs/cubefs-dashboard/backend/model"
)

// Runner coordinates the lifecycle of a single posix-check run end-to-end:
// build the Job manifest, hand it to the K8s API, watch until terminal,
// pull pod logs, parse the TAP stream, persist results to MySQL.
//
// One Runner instance is shared across the process; each run gets its own
// goroutine via Submit().
type Runner struct {
	client    *Client
	namespace string

	mu       sync.Mutex
	inflight map[int64]chan struct{} // runID → cancel signal
}

// SubmitParams is what the HTTP handler captures from the user.
type SubmitParams struct {
	ClusterName string
	TargetVol   string
	MountSubdir string
	SuiteImage  string
	TestFilter  string
	TriggerUser string
}

var (
	defaultNamespace = "cfs-monitor"
	defaultImage     = "hub.shiyak-office.com/storage/pjd-fstest:20090130-rc2"
)

// NewRunner constructs a Runner using the in-cluster ServiceAccount.
// Returns an error if not running in a pod (token / CA files missing).
func NewRunner(namespace string) (*Runner, error) {
	if namespace == "" {
		namespace = defaultNamespace
	}
	cli, err := NewInClusterClient(namespace)
	if err != nil {
		return nil, err
	}
	return &Runner{
		client:    cli,
		namespace: namespace,
		inflight:  make(map[int64]chan struct{}),
	}, nil
}

// Submit creates the DB row, kicks off the K8s Job and returns immediately
// with the new run id. The actual watching and result parsing happens in a
// background goroutine.
func (r *Runner) Submit(p SubmitParams) (*model.PosixCheckRun, error) {
	if p.SuiteImage == "" {
		p.SuiteImage = defaultImage
	}
	masterAddr, err := resolveMasterAddr(p.ClusterName)
	if err != nil {
		return nil, fmt.Errorf("resolve master addr: %w", err)
	}

	run := &model.PosixCheckRun{
		ClusterName: p.ClusterName,
		TargetVol:   p.TargetVol,
		MountSubdir: p.MountSubdir,
		SuiteImage:  p.SuiteImage,
		TestFilter:  p.TestFilter,
		TriggerUser: p.TriggerUser,
		Status:      string(model.PosixCheckRunPending),
		CreatedAt:   time.Now(),
	}
	if err := model.CreatePosixCheckRun(run); err != nil {
		return nil, fmt.Errorf("create run record: %w", err)
	}

	if run.MountSubdir == "" {
		run.MountSubdir = DefaultSubdir(run.Id)
	}
	jobName := JobNameFor(p.ClusterName, run.Id)
	run.K8sJobName = jobName

	manifest, err := BuildJobManifest(JobSpecParams{
		RunID:       run.Id,
		Namespace:   r.namespace,
		JobName:     jobName,
		SuiteImage:  p.SuiteImage,
		TargetVol:   p.TargetVol,
		MasterAddr:  masterAddr,
		MountSubdir: run.MountSubdir,
		TestFilter:  p.TestFilter,
	})
	if err != nil {
		r.failRun(run.Id, fmt.Sprintf("build job manifest: %v", err))
		return run, err
	}

	if _, err := r.client.CreateJob(manifest); err != nil {
		r.failRun(run.Id, fmt.Sprintf("create k8s job: %v", err))
		return run, err
	}

	_ = model.UpdatePosixCheckRun(run.Id, map[string]interface{}{
		"k8s_job_name": jobName,
		"mount_subdir": run.MountSubdir,
		"status":       string(model.PosixCheckRunRunning),
	})
	run.Status = string(model.PosixCheckRunRunning)

	cancel := make(chan struct{})
	r.mu.Lock()
	r.inflight[run.Id] = cancel
	r.mu.Unlock()
	go r.watch(run.Id, jobName, cancel)

	return run, nil
}

// Cancel marks an in-flight run cancelled and deletes its K8s Job. Safe to
// call when the run has already completed.
func (r *Runner) Cancel(runId int64) error {
	run, err := model.GetPosixCheckRun(runId)
	if err != nil {
		return err
	}
	if run.K8sJobName != "" {
		_ = r.client.DeleteJob(run.K8sJobName)
	}
	r.mu.Lock()
	if c, ok := r.inflight[runId]; ok {
		close(c)
		delete(r.inflight, runId)
	}
	r.mu.Unlock()
	return model.UpdatePosixCheckRun(runId, map[string]interface{}{
		"status":      string(model.PosixCheckRunCancelled),
		"finished_at": time.Now(),
	})
}

// Delete removes a run record and its failure rows from the database. If
// the run is still in flight the K8s Job is killed and the watcher is
// signalled before the rows are deleted (otherwise the watcher would race
// and re-create rows or write back final status to a deleted record).
func (r *Runner) Delete(runId int64) error {
	run, err := model.GetPosixCheckRun(runId)
	if err != nil {
		return err
	}
	// Stop the K8s Job if still alive; ignore error (Job may already be gone
	// or never existed if Submit failed at create-time).
	if run.K8sJobName != "" {
		_ = r.client.DeleteJob(run.K8sJobName)
	}
	r.mu.Lock()
	if c, ok := r.inflight[runId]; ok {
		close(c)
		delete(r.inflight, runId)
	}
	r.mu.Unlock()
	return model.DeletePosixCheckRun(runId)
}

// watch polls the Job status until terminal, pulls pod logs, parses TAP
// and persists pass/fail counts + failure rows. Cancelled runs short-circuit.
func (r *Runner) watch(runId int64, jobName string, cancel <-chan struct{}) {
	t0 := time.Now()
	defer func() {
		r.mu.Lock()
		delete(r.inflight, runId)
		r.mu.Unlock()
	}()

	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-cancel:
			return
		case <-tick.C:
			st, err := r.client.GetJobStatus(jobName)
			if err != nil {
				// transient errors are fine; keep polling. Hard failures
				// (Job deleted out-of-band) eventually time out via
				// activeDeadlineSeconds set in the manifest.
				continue
			}
			if !st.Done {
				continue
			}
			logs, lerr := r.client.PodLogsForJob(jobName)
			if lerr != nil {
				r.failRun(runId, fmt.Sprintf("fetch pod logs: %v", lerr))
				return
			}
			parsed := ParseTAP(logs)
			for i := range parsed.Failures {
				parsed.Failures[i].RunId = runId
			}
			if err := model.BatchInsertPosixCheckFailures(parsed.Failures); err != nil {
				r.failRun(runId, fmt.Sprintf("insert failures: %v", err))
				return
			}
			status := model.PosixCheckRunDone
			if st.Phase == "failed" && parsed.TotalCount == 0 {
				// Job itself crashed before producing TAP. Mark failed.
				status = model.PosixCheckRunFailed
			}
			_ = model.UpdatePosixCheckRun(runId, map[string]interface{}{
				"status":       string(status),
				"pass_count":   parsed.PassCount,
				"fail_count":   parsed.FailCount,
				"skip_count":   parsed.SkipCount,
				"total_count":  parsed.TotalCount,
				"duration_sec": int(time.Since(t0).Seconds()),
				"finished_at":  time.Now(),
			})
			return
		}
	}
}

func (r *Runner) failRun(runId int64, msg string) {
	_ = model.UpdatePosixCheckRun(runId, map[string]interface{}{
		"status":      string(model.PosixCheckRunFailed),
		"error_msg":   msg,
		"finished_at": time.Now(),
	})
}

// resolveMasterAddr returns the master peer list of the named cluster from
// the clusters table, joined as "host1:17010,host2:17010,...".
func resolveMasterAddr(clusterName string) (string, error) {
	c := &model.Cluster{}
	row, err := c.FindName(clusterName)
	if err != nil {
		return "", err
	}
	if row == nil || len(row.MasterAddr) == 0 {
		return "", fmt.Errorf("cluster %q has no master_addr configured", clusterName)
	}
	return strings.Join(row.MasterAddr, ","), nil
}

func joinHostList(hosts []string) string {
	return strings.Join(hosts, ",")
}
