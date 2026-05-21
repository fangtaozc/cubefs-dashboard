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
	"regexp"
	"strconv"
	"strings"

	"github.com/cubefs/cubefs-dashboard/backend/model"
)

// ParseResult is the aggregated outcome of one pjd-fstest run, derived
// from the raw TAP stream captured from the pod's stdout.
type ParseResult struct {
	PassCount  int
	FailCount  int
	SkipCount  int
	TotalCount int
	Failures   []model.PosixCheckFailure
}

// ParseTAP extracts per-test results from the prove(1) TAP stream.
//
// pjd-fstest's `prove -r` output looks like:
//
//	/opt/pjd-fstest/tests/chmod/00.t .. ok
//	/opt/pjd-fstest/tests/chmod/01.t .. 1/5
//	not ok 3 - chmod 0777 returns EACCES
//	# Failed test (chmod/01.t at line 12)
//	#   got: 0 EPERM
//	#   expected: 0 EACCES
//	ok 4 - chmod 0644 returns 0
//	ok 5 # SKIP not applicable
//	/opt/pjd-fstest/tests/chmod/01.t .. Failed 1/5 subtests
//
// The parser walks line-by-line, tracking the current test file from the
// header line (".. ok" or ".. NN/MM") and capturing every `not ok` line
// (with optional "# SKIP" / "# TODO" markers) as a failure. Comment lines
// prefixed `#   got:` / `#   expected:` are stitched onto the most recent
// not-ok entry.
func ParseTAP(raw string) ParseResult {
	res := ParseResult{Failures: []model.PosixCheckFailure{}}

	currentFile := ""
	var lastFailure *model.PosixCheckFailure

	// "/path/to/test.t .. ok" or  "/path/to/test.t .. 1/5"
	// Leading "[HH:MM:SS]" timestamp from prove --timer is tolerated.
	// Trailing dots may be any count ("..", "....", "......") — prove pads
	// to align test paths.
	reHeader := regexp.MustCompile(`(?:^|\s)(\S+\.t)\s+\.{2,}\s*`)
	// "ok 3 - description"  or  "ok 3 # SKIP reason"
	reOk := regexp.MustCompile(`^ok\s+(\d+)(?:\s+(?:-\s+)?(.+))?$`)
	// "not ok 3 - description"
	reNotOk := regexp.MustCompile(`^not ok\s+(\d+)(?:\s+(?:-\s+)?(.+))?$`)
	// "#   got: 0 EPERM"  /  "#   expected: 0 EACCES"
	reGot := regexp.MustCompile(`^#\s+got:\s+(.+)$`)
	reExp := regexp.MustCompile(`^#\s+expected:\s+(.+)$`)
	// pjd-fstest doesn't print TAP "- description" or "got/expected" comments;
	// instead its .t files do `bash -x` + helpers like `expect` / `test_check`
	// + final `echo 'not ok N'`. The meaningful context for each failure is
	// the bash trace lines that ran right before the `not ok`. We keep a
	// rolling buffer (any number of '+' prefixes — bash -x uses '++ ' for
	// nested subshells) and fold it into Description on each failure.
	reTrace := regexp.MustCompile(`^\++\s+(.+)$`)
	// "+ expect 0 chmod foo 0644" / "+ expect EACCES chmod foo 0644" —
	// pjd misc.sh helper that explicitly takes (expected_return, syscall args).
	reExpectCmd := regexp.MustCompile(`^expect\s+(\S+)\s+(.+)$`)
	// "+ test_check 1779243979 -lt 1779243979" — pjd misc.sh helper that
	// takes (actual_value, comparison_op, expected_value).
	reTestCheck := regexp.MustCompile(`^test_check\s+(\S+)\s+(\S+)\s+(\S+)\s*$`)
	const traceWindow = 8
	traceBuf := make([]string, 0, traceWindow)
	pushTrace := func(s string) {
		if len(traceBuf) >= traceWindow {
			traceBuf = traceBuf[1:]
		}
		traceBuf = append(traceBuf, s)
	}
	// Trace lines we don't want surfaced as description noise.
	isNoise := func(s string) bool {
		if strings.HasPrefix(s, "echo 'not ok") || strings.HasPrefix(s, "echo 'ok") {
			return true
		}
		// `var=value` assignments by themselves carry little signal.
		if idx := strings.IndexAny(s, " "); idx == -1 && strings.Contains(s, "=") {
			return true
		}
		return false
	}

	for _, raw := range strings.Split(raw, "\n") {
		line := strings.TrimRight(raw, "\r")
		if m := reHeader.FindStringSubmatch(line); m != nil {
			currentFile = trimTestPrefix(m[1])
			lastFailure = nil
			traceBuf = traceBuf[:0]
			continue
		}
		// Capture bash -x trace lines into the rolling buffer — they form the
		// description / expected for the next `not ok`.
		if m := reTrace.FindStringSubmatch(line); m != nil {
			pushTrace(strings.TrimSpace(m[1]))
			continue
		}
		if m := reOk.FindStringSubmatch(line); m != nil {
			desc := strings.TrimSpace(m[2])
			if strings.Contains(strings.ToUpper(desc), "# SKIP") || strings.Contains(strings.ToUpper(desc), "#SKIP") {
				res.SkipCount++
			} else {
				res.PassCount++
			}
			res.TotalCount++
			lastFailure = nil
			traceBuf = traceBuf[:0]
			continue
		}
		if m := reNotOk.FindStringSubmatch(line); m != nil {
			num, _ := strconv.Atoi(m[1])
			desc := strings.TrimSpace(m[2])
			f := model.PosixCheckFailure{
				TestFile:    currentFile,
				TestNumber:  num,
				Description: truncateField(desc, 500),
				Syscall:     guessSyscall(currentFile),
			}
			// If pjd didn't print a TAP "- description", synthesize from the
			// bash trace window. Prefer structured helpers (expect / test_check)
			// to extract expected/actual; fall back to the last few non-noise
			// trace lines joined as Description.
			if f.Description == "" && len(traceBuf) > 0 {
				// Look for the most recent expect / test_check call.
				for i := len(traceBuf) - 1; i >= 0; i-- {
					if em := reExpectCmd.FindStringSubmatch(traceBuf[i]); em != nil {
						f.Expected = truncateField(em[1], 500)
						f.Description = truncateField("expect "+em[1]+" "+em[2], 500)
						// Last bracket expression `[ a OP b ]` shows the
						// actual value that failed the comparison.
						for j := len(traceBuf) - 1; j > i; j-- {
							if strings.HasPrefix(traceBuf[j], "'[' ") || strings.HasPrefix(traceBuf[j], "[ ") {
								f.Actual = truncateField(traceBuf[j], 500)
								break
							}
						}
						break
					}
					if tm := reTestCheck.FindStringSubmatch(traceBuf[i]); tm != nil {
						f.Description = truncateField("test_check "+tm[1]+" "+tm[2]+" "+tm[3], 500)
						f.Actual = truncateField(tm[1], 500)
						f.Expected = truncateField(tm[3], 500)
						break
					}
				}
				if f.Description == "" {
					// Generic fallback: last 3 non-noise lines joined.
					parts := make([]string, 0, 3)
					for i := len(traceBuf) - 1; i >= 0 && len(parts) < 3; i-- {
						if !isNoise(traceBuf[i]) {
							parts = append([]string{traceBuf[i]}, parts...)
						}
					}
					f.Description = truncateField(strings.Join(parts, " · "), 500)
				}
			}
			res.Failures = append(res.Failures, f)
			lastFailure = &res.Failures[len(res.Failures)-1]
			res.FailCount++
			res.TotalCount++
			traceBuf = traceBuf[:0]
			continue
		}
		if lastFailure != nil {
			if m := reGot.FindStringSubmatch(line); m != nil {
				lastFailure.Actual = truncateField(m[1], 500)
				continue
			}
			if m := reExp.FindStringSubmatch(line); m != nil {
				lastFailure.Expected = truncateField(m[1], 500)
				continue
			}
		}
	}
	return res
}

// trimTestPrefix drops the leading "/opt/pjd-fstest/tests/" prefix if it's
// present so the recorded test_file is e.g. "rename/00.t" not the full
// absolute path inside the container.
func trimTestPrefix(path string) string {
	const prefix = "/opt/pjd-fstest/tests/"
	if strings.HasPrefix(path, prefix) {
		return path[len(prefix):]
	}
	// Generic fallback: keep the last two path segments (dir/file.t).
	parts := strings.Split(path, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "/" + parts[len(parts)-1]
	}
	return path
}

// guessSyscall extracts the syscall hint from the test file's parent dir.
// pjd-fstest organises tests under tests/<syscall>/NN.t so e.g.
// "rename/00.t" → "rename".
func guessSyscall(testFile string) string {
	if testFile == "" {
		return ""
	}
	parts := strings.SplitN(testFile, "/", 2)
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

func truncateField(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
