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
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// JobSpecParams is everything the dashboard collects from the user before
// composing a one-shot K8s Job manifest.
type JobSpecParams struct {
	RunID       int64
	Namespace   string // K8s namespace (typically cfs-monitor)
	JobName     string // unique per run, e.g. posix-check-42
	SuiteImage  string // pjd-fstest image to run
	TargetVol   string // CubeFS volume name to mount
	MasterAddr  string // CubeFS master addresses, comma-separated host:port
	MountSubdir string // subdir under /mnt/target for this run
	TestFilter  string // optional space-separated test paths
}

// BuildJobManifest renders a K8s Job spec for one posix-check run. The Pod
// mounts the target CubeFS volume via the CSI driver into /mnt/target,
// runs as root (the pjd tests need it to exercise chown / setuid bits)
// and sets ttlSecondsAfterFinished so completed Jobs are reaped.
//
// TTL is 1800s (30 min) to leave enough time for the dashboard to fetch
// pod logs even if a worker restart delays the watcher.
func BuildJobManifest(p JobSpecParams) ([]byte, error) {
	if p.JobName == "" || p.SuiteImage == "" || p.TargetVol == "" || p.MasterAddr == "" {
		return nil, fmt.Errorf("BuildJobManifest: missing required field")
	}
	priv := true
	root := int64(0)
	ttl := int32(1800)
	deadline := int64(3600)
	backoff := int32(0)

	manifest := map[string]interface{}{
		"apiVersion": "batch/v1",
		"kind":       "Job",
		"metadata": map[string]interface{}{
			"name":      p.JobName,
			"namespace": p.Namespace,
			"labels": map[string]string{
				"app":    "posix-check",
				"run-id": fmt.Sprintf("%d", p.RunID),
			},
		},
		"spec": map[string]interface{}{
			"ttlSecondsAfterFinished": ttl,
			"activeDeadlineSeconds":   deadline,
			"backoffLimit":            backoff,
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]string{
						"app":    "posix-check",
						"run-id": fmt.Sprintf("%d", p.RunID),
					},
				},
				"spec": map[string]interface{}{
					"restartPolicy": "Never",
					// Pin the pod to a node where the CubeFS CSI driver is
					// running (matches the cubefs.io/csi=true label set by
					// cubefs-deploy/_envcommon node-labels). Without this
					// k8s may pick a node where the CSI driver is not
					// registered and the CSI volume mount fails.
					"nodeSelector": map[string]string{
						"cubefs.io/csi": "true",
					},
					"containers": []map[string]interface{}{
						{
							"name":  "pjd",
							"image": p.SuiteImage,
							"env": []map[string]interface{}{
								{"name": "MOUNT_DIR", "value": "/mnt/target"},
								{"name": "SUBDIR", "value": p.MountSubdir},
								{"name": "TEST_FILTER", "value": p.TestFilter},
							},
							"securityContext": map[string]interface{}{
								"privileged": priv,
								"runAsUser":  root,
							},
							"volumeMounts": []map[string]interface{}{
								{"name": "target", "mountPath": "/mnt/target"},
							},
						},
					},
					"volumes": []map[string]interface{}{
						{
							"name": "target",
							// Generic Ephemeral Volume → bound PVC provisioned from
							// the CubeFS StorageClass. We use this rather than CSI
							// inline ephemeral because CubeFS CSI v3 NodePublish
							// expects a non-empty stagingTargetPath (the inline
							// path skips NodeStage), making inline mounts fail.
							// Each run gets a fresh CubeFS volume — pjd-fstest
							// validates POSIX semantics, not specific volume data,
							// so an ephemeral fresh volume is fine.
							"ephemeral": map[string]interface{}{
								"volumeClaimTemplate": map[string]interface{}{
									"metadata": map[string]interface{}{
										"labels": map[string]string{
											"app":    "posix-check",
											"run-id": fmt.Sprintf("%d", p.RunID),
										},
									},
									"spec": map[string]interface{}{
										"accessModes":      []string{"ReadWriteOnce"},
										"storageClassName": "cfs-sc",
										"resources": map[string]interface{}{
											"requests": map[string]string{
												"storage": "1Gi",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return json.Marshal(manifest)
}

// SanitizeName produces a DNS-1123-compliant job name segment from arbitrary
// input. Used to derive Job names from cluster + run id while keeping the
// total under 63 chars (the K8s name limit).
func SanitizeName(s string) string {
	s = strings.ToLower(s)
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			out = append(out, c)
		} else {
			out = append(out, '-')
		}
	}
	r := strings.Trim(string(out), "-")
	if len(r) > 40 {
		r = r[:40]
	}
	return r
}

// JobNameFor builds a unique Job name for a (cluster, runId) pair.
func JobNameFor(cluster string, runId int64) string {
	return fmt.Sprintf("posix-check-%s-%d", SanitizeName(cluster), runId)
}

// DefaultSubdir creates a per-run subdir name under the mount root.
// Uses runId + timestamp so concurrent runs against the same volume
// don't collide.
func DefaultSubdir(runId int64) string {
	return fmt.Sprintf("pjd-%d-%d", runId, time.Now().Unix())
}
