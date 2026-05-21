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

// Package posixcheck implements the POSIX compliance feature: launching
// one-shot pjd-fstest K8s Jobs, watching them, parsing TAP output and
// recording results into MySQL.
package posixcheck

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Minimal Kubernetes REST client. Scope is intentionally narrow: create a
// Job, query its status, fetch a single pod's logs, delete the Job. The
// dashboard pod must run with a ServiceAccount that has the matching RBAC
// (Jobs: create/get/list/delete, Pods: list/get/log).
//
// We avoid importing k8s.io/client-go to keep go.mod / image size lean.

const (
	defaultAPIHost   = "kubernetes.default.svc"
	defaultTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	defaultCAPath    = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
)

// Client is a tiny K8s REST client used by the POSIX compliance feature.
type Client struct {
	apiHost   string
	token     string
	http      *http.Client
	namespace string
}

// NewInClusterClient builds a Client from the in-cluster ServiceAccount.
// It reads the bearer token and CA bundle from the well-known paths the
// kubelet projects into every Pod. Returns an error if those files are
// missing (e.g. when running outside a cluster).
func NewInClusterClient(namespace string) (*Client, error) {
	tokenBytes, err := os.ReadFile(defaultTokenPath)
	if err != nil {
		return nil, fmt.Errorf("read service account token: %w", err)
	}
	caBytes, err := os.ReadFile(defaultCAPath)
	if err != nil {
		return nil, fmt.Errorf("read service account CA: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caBytes) {
		return nil, errors.New("failed to parse service account CA bundle")
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
	}
	if namespace == "" {
		namespace = "cfs-monitor"
	}
	return &Client{
		apiHost:   defaultAPIHost,
		token:     string(bytes.TrimSpace(tokenBytes)),
		http:      &http.Client{Transport: tr, Timeout: 30 * time.Second},
		namespace: namespace,
	}, nil
}

func (c *Client) do(method, path string, body []byte) ([]byte, int, error) {
	url := fmt.Sprintf("https://%s%s", c.apiHost, path)
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		return b, resp.StatusCode, fmt.Errorf("k8s api %s %s: %d %s", method, path, resp.StatusCode, truncate(string(b), 400))
	}
	return b, resp.StatusCode, nil
}

// CreateJob posts the given Job manifest (JSON) and returns the created
// Job's name (echoed in the API response) so callers can reuse it for
// status/log lookups.
func (c *Client) CreateJob(manifestJSON []byte) (string, error) {
	path := fmt.Sprintf("/apis/batch/v1/namespaces/%s/jobs", c.namespace)
	resp, _, err := c.do("POST", path, manifestJSON)
	if err != nil {
		return "", err
	}
	var out struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(resp, &out); err != nil {
		return "", fmt.Errorf("decode job response: %w", err)
	}
	return out.Metadata.Name, nil
}

// JobStatus is a slimmed-down view of batch/v1 Job status used by Watch.
type JobStatus struct {
	Active    int  // pods still running
	Succeeded int  // pods completed with exit 0
	Failed    int  // pods with non-zero exit
	Done      bool // succeeded > 0 || failed > 0 (terminal)
	Phase     string
}

// GetJobStatus fetches the named Job and projects its status into JobStatus.
func (c *Client) GetJobStatus(name string) (*JobStatus, error) {
	path := fmt.Sprintf("/apis/batch/v1/namespaces/%s/jobs/%s", c.namespace, name)
	resp, _, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var raw struct {
		Status struct {
			Active    int `json:"active"`
			Succeeded int `json:"succeeded"`
			Failed    int `json:"failed"`
		} `json:"status"`
	}
	if err := json.Unmarshal(resp, &raw); err != nil {
		return nil, fmt.Errorf("decode job status: %w", err)
	}
	st := &JobStatus{
		Active:    raw.Status.Active,
		Succeeded: raw.Status.Succeeded,
		Failed:    raw.Status.Failed,
	}
	if st.Succeeded > 0 {
		st.Done, st.Phase = true, "succeeded"
	} else if st.Failed > 0 {
		st.Done, st.Phase = true, "failed"
	} else if st.Active > 0 {
		st.Phase = "running"
	} else {
		st.Phase = "pending"
	}
	return st, nil
}

// PodLogsForJob locates the (typically single) pod created for the given
// Job and returns its full stdout/stderr stream as a string.
func (c *Client) PodLogsForJob(jobName string) (string, error) {
	listPath := fmt.Sprintf("/api/v1/namespaces/%s/pods?labelSelector=job-name=%s", c.namespace, jobName)
	resp, _, err := c.do("GET", listPath, nil)
	if err != nil {
		return "", err
	}
	var lst struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
		} `json:"items"`
	}
	if err := json.Unmarshal(resp, &lst); err != nil {
		return "", fmt.Errorf("decode pod list: %w", err)
	}
	if len(lst.Items) == 0 {
		return "", fmt.Errorf("no pods found for job %q", jobName)
	}
	podName := lst.Items[0].Metadata.Name
	logPath := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/log", c.namespace, podName)
	body, _, err := c.do("GET", logPath, nil)
	return string(body), err
}

// DeleteJob removes the Job (and its pods via cascading delete).
func (c *Client) DeleteJob(name string) error {
	path := fmt.Sprintf("/apis/batch/v1/namespaces/%s/jobs/%s?propagationPolicy=Background", c.namespace, name)
	_, _, err := c.do("DELETE", path, nil)
	return err
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
