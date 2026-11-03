package hosts

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"ydb-installer/internal/models"
	"ydb-installer/internal/services/sshclient"
)

// All per-host checks run in a single combined SSH session to avoid the
// flakiness of opening 5-10 separate connections per host: a transient network
// blip on just one connection would cause a partial result (e.g. "SSH ok" but
// "Python not found") even though the host is healthy. The combined session is
// also retried once on failure.

type Checker struct{}

func NewChecker() *Checker {
	return &Checker{}
}

// NTPOutputs holds the raw output text for each NTP detection command run
// during the combined single-SSH-session host check. Each field may be
// empty if the corresponding command produced no output (e.g. the tool is
// not installed on the remote host).
type NTPOutputs struct {
	TimesyncStatus   string
	ShowTimesync     string
	PlainTimedatectl string
	ChronycTracking  string
	NtpqPeers        string
	SystemctlActive  string
}

const (
	markerSSHOK        = "###SSH_OK###"
	markerPython       = "###PYTHON###"
	markerCPU          = "###CPU###"
	markerPorts        = "###PORTS###"
	markerNTPTimesync  = "###NTP_TIMESYNC###"
	markerNTPShow      = "###NTP_SHOW###"
	markerNTPPlain     = "###NTP_PLAIN###"
	markerNTPChronyc   = "###NTP_CHRONYC###"
	markerNTPNtpq      = "###NTP_NTPQ###"
	markerNTPSystemctl = "###NTP_SYSTEMCTL###"
	markerEnd          = "###END###"
)

// buildCheckScript returns the POSIX-sh script executed on the remote host.
// It is intentionally written in portable /bin/sh syntax (no bashisms)
// since we invoke it via `ssh ... sh -s` with the script piped on stdin.
func buildCheckScript() string {
	lines := []string{
		"echo '" + markerSSHOK + "'",
		"echo '" + markerPython + "'",
		// Python < 3.4 prints its version to stderr instead of stdout.
		"{ python3 --version; } 2>&1 || { python --version; } 2>&1 || echo 'PYTHON_NOT_FOUND'",
		"echo '" + markerCPU + "'",
		"nproc 2>/dev/null || grep -c '^processor' /proc/cpuinfo 2>/dev/null || echo 1",
		"echo '" + markerPorts + "'",
		"ss -tlnp 2>/dev/null || netstat -tlnp 2>/dev/null || true",
		"echo '" + markerNTPTimesync + "'",
		"timedatectl timesync-status 2>/dev/null || true",
		"echo '" + markerNTPShow + "'",
		"timedatectl show-timesync --all 2>/dev/null || true",
		"echo '" + markerNTPPlain + "'",
		"timedatectl 2>/dev/null || true",
		"echo '" + markerNTPChronyc + "'",
		"chronyc tracking 2>/dev/null || true",
		"echo '" + markerNTPNtpq + "'",
		"ntpq -p 2>/dev/null || true",
		"echo '" + markerNTPSystemctl + "'",
		"for svc in chronyd.service ntp.service ntpd.service systemd-timesyncd.service; do",
		"  st=$(systemctl is-active \"$svc\" 2>/dev/null)",
		"  [ \"$st\" = \"active\" ] && echo \"$svc:active\"",
		"done",
		"echo '" + markerEnd + "'",
	}
	return strings.Join(lines, "\n") + "\n"
}

func parseCheckScriptOutput(stdout string) map[string]string {
	markers := []string{
		markerSSHOK, markerPython, markerCPU, markerPorts,
		markerNTPTimesync, markerNTPShow, markerNTPPlain,
		markerNTPChronyc, markerNTPNtpq, markerNTPSystemctl, markerEnd,
	}
	markerSet := make(map[string]bool, len(markers))
	for _, m := range markers {
		markerSet[m] = true
	}

	sections := make(map[string]string)
	current := ""
	var buf strings.Builder
	for _, line := range strings.Split(stdout, "\n") {
		trimmed := strings.TrimSpace(line)
		if markerSet[trimmed] {
			if current != "" {
				sections[current] = buf.String()
			}
			current = trimmed
			buf.Reset()
			continue
		}
		if current != "" {
			buf.WriteString(line)
			buf.WriteString("\n")
		}
	}
	if current != "" {
		sections[current] = buf.String()
	}
	return sections
}

// runSSHScript pipes the script via stdin to `sh -s` on the remote host,
// avoiding shell-quoting issues that would arise from passing it as a
// command-line argument.
func runSSHScript(sshArgs []string, script string, timeout time.Duration) (stdout string, stderr string, err error) {
	cmdArgs := append(append([]string{}, sshArgs...), "sh", "-s")
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ssh", cmdArgs...)
	cmd.Env = append(os.Environ(), "LANG=C")
	cmd.Stdin = strings.NewReader(script)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err = cmd.Run()
	return stdoutBuf.String(), stderrBuf.String(), err
}

func (c *Checker) CheckHosts(req *models.CheckHostsRequest) (*models.CheckHostsResponse, error) {
	if len(req.Hosts) == 0 {
		return &models.CheckHostsResponse{Results: []models.CheckHostResult{}}, nil
	}

	if req.SSH != nil {
		if err := sshclient.ValidateKeyPath(req.SSH.KeyPath); err != nil {
			return nil, err
		}
		if req.SSH.UseJumpHost && req.SSH.JumpHostFQDN == "" {
			return nil, fmt.Errorf("Jump host FQDN is required when use_jump_host is true")
		}
	}

	results := make([]models.CheckHostResult, len(req.Hosts))
	var wg sync.WaitGroup

	for i, host := range req.Hosts {
		wg.Add(1)
		go func(idx int, h string) {
			defer wg.Done()
			results[idx] = c.checkHost(h, req.SSH, req.Ports)
		}(i, host)
	}

	wg.Wait()

	summary := c.measureDrift(req.Hosts, req.SSH, results)

	return &models.CheckHostsResponse{Results: results, TimeSync: summary}, nil
}

// measureDrift computes the maximum time drift between hosts using the NTP
// offset values collected during checkHost. This is far more accurate than
// running `date +%s%N` via SSH, because the NTP offset is measured by the local
// NTP daemon against its upstream server — it is not affected by SSH
// connection latency or goroutine scheduling jitter.
//
// Drift = max(offsets) - min(offsets). Returns nil if fewer than 2 hosts have
// valid NTP offset data.
func (c *Checker) measureDrift(hosts []string, ssh *models.SSHConfig, results []models.CheckHostResult) *models.TimeSyncSummary {
	summary := &models.TimeSyncSummary{}
	summary.Threshold.WarningMs = 5
	summary.Threshold.ErrorMs = 25

	// If SSH failed on a host, treat NTP as unverifiable (= missing),
	// otherwise all-SSH-failure would incorrectly produce "ok" status with 0 drift.
	ntpMissing := false
	ntpNotSynced := false
	for _, r := range results {
		if !r.SSH || !r.TimeSync.NTPActive {
			ntpMissing = true
			continue
		}
		if r.TimeSync.NTPActive && !r.TimeSync.NTPSynced {
			ntpNotSynced = true
		}
	}

	if ntpMissing {
		summary.MaxDriftMs = 0
		summary.Status = "error"
		summary.NTPNotSynced = ntpNotSynced
		return summary
	}

	var offsets []int64
	for _, r := range results {
		if r.SSH && r.TimeSync.NTPActive {
			offsets = append(offsets, r.TimeSync.ClockOffsetMs)
		}
	}

	if len(offsets) < 2 {
		summary.MaxDriftMs = 0
		if ntpNotSynced {
			summary.Status = "warning"
		} else {
			summary.Status = "ok"
		}
		summary.NTPNotSynced = ntpNotSynced
		return summary
	}

	var minOffset, maxOffset int64 = offsets[0], offsets[0]
	for _, off := range offsets[1:] {
		if off < minOffset {
			minOffset = off
		}
		if off > maxOffset {
			maxOffset = off
		}
	}

	drift := maxOffset - minOffset
	if drift < 0 {
		drift = -drift
	}
	summary.MaxDriftMs = drift

	switch {
	case summary.MaxDriftMs >= summary.Threshold.ErrorMs:
		summary.Status = "error"
	case summary.MaxDriftMs >= summary.Threshold.WarningMs:
		summary.Status = "warning"
	case ntpNotSynced:
		summary.Status = "warning"
	default:
		summary.Status = "ok"
	}

	summary.NTPNotSynced = ntpNotSynced

	return summary
}

// checkHost performs all checks (SSH, Python, CPU, ports, NTP) on a single
// host using one combined SSH session, retried once on failure to absorb
// transient network blips (DNS hiccup, TCP reset, slow handshake).
func (c *Checker) checkHost(host string, ssh *models.SSHConfig, ports []int) models.CheckHostResult {
	result := models.CheckHostResult{
		Host: host,
	}

	sshArgs := sshclient.BuildSSHArgs(host, ssh)

	fmt.Printf("[DEBUG] SSH command: ssh %v\n", sshArgs)

	script := buildCheckScript()
	const sessionTimeout = 25 * time.Second

	var stdout, stderr string
	var err error

	const maxAttempts = 2
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		stdout, stderr, err = runSSHScript(sshArgs, script, sessionTimeout)
		if err == nil && strings.Contains(stdout, markerSSHOK) {
			break
		}
		if attempt < maxAttempts {
			time.Sleep(1 * time.Second)
		}
	}

	if err != nil || !strings.Contains(stdout, markerSSHOK) {
		result.SSH = false
		errMsg := stderr
		if errMsg == "" && err != nil {
			errMsg = err.Error()
		}
		result.SSHError = fmt.Sprintf("SSH error: %s", c.cleanSSHError(errMsg, err))
		result.TimeSync = models.TimeSyncResult{
			NTPActive: false,
			Error:     "SSH connectivity required for NTP check",
		}
		return result
	}
	result.SSH = true

	sections := parseCheckScriptOutput(stdout)

	pythonOut := strings.TrimSpace(sections[markerPython])
	if pythonOut == "" || pythonOut == "PYTHON_NOT_FOUND" {
		result.Python = false
		result.PythonError = "Python not found"
		return result
	}

	re := regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(pythonOut)
	if len(matches) < 4 {
		result.Python = true
		result.PythonVer = pythonOut
		return result
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	versionNum := major*10000 + minor*100 + patch
	versionDisplay := fmt.Sprintf("%d.%d.%d", major, minor, patch)

	const minVersion = 30800
	if versionNum < minVersion {
		result.Python = false
		result.PythonError = fmt.Sprintf("Python %s is too old (requires ≥ 3.8)", versionDisplay)
		return result
	}

	result.Python = true
	result.PythonVer = versionDisplay

	cpuCores, cpuErr := strconv.Atoi(strings.TrimSpace(sections[markerCPU]))
	if cpuErr != nil || cpuCores <= 0 {
		result.CPUCores = 1
	} else {
		result.CPUCores = cpuCores
	}

	portsToCheck := ports
	if len(portsToCheck) == 0 {
		portsToCheck = []int{2135, 8765}
	}
	portDetails, allFree, portErrors := checkPortsAvailability(sections[markerPorts], portsToCheck)
	result.Ports = allFree
	result.PortDetails = portDetails
	if !allFree {
		result.PortsError = strings.Join(portErrors, "; ")
	}

	result.TimeSync = c.CheckNTPStatus(NTPOutputs{
		TimesyncStatus:   sections[markerNTPTimesync],
		ShowTimesync:     sections[markerNTPShow],
		PlainTimedatectl: sections[markerNTPPlain],
		ChronycTracking:  sections[markerNTPChronyc],
		NtpqPeers:        sections[markerNTPNtpq],
		SystemctlActive:  sections[markerNTPSystemctl],
	})

	return result
}

// CheckNTPStatus determines whether an NTP daemon (chrony / systemd-timesyncd
// / ntpd) is running and synchronized on the remote host, using the raw
// output already collected during the combined single-SSH-session check.
// Detection is attempted in order of reliability, falling back to broader
// checks when a more specific one yields nothing:
//
//  1. timedatectl timesync-status        (human-readable, has Offset: field)
//  2. timedatectl show-timesync --all    (key=value format, needs systemd ≥219)
//  3. timedatectl                        (plain status, wider compat)
//  4. chronyc tracking                   (chrony-specific)
//  5. ntpq -p                           (ntpd-specific)
//  6. systemctl is-active <service>      (broadest, direct service check)
func (c *Checker) CheckNTPStatus(out NTPOutputs) models.TimeSyncResult {
	if out.TimesyncStatus != "" {
		if result := c.ParseTimedatectlTimesync(out.TimesyncStatus); result.NTPActive {
			return result
		}
	}

	if out.ShowTimesync != "" {
		if result := c.ParseTimedatectl(out.ShowTimesync); result.NTPActive {
			return result
		}
	}

	// Some systems have timedatectl but lack the `show-timesync` subcommand or
	// output fields like NTPService= / NTPSynchronized= which our parser expects.
	if out.PlainTimedatectl != "" {
		if result := c.ParseTimedatectlStatus(out.PlainTimedatectl); result.NTPActive {
			return result
		}
	}

	if out.ChronycTracking != "" {
		if result := c.ParseChronyc(out.ChronycTracking); result.NTPActive {
			return result
		}
	}

	if out.NtpqPeers != "" {
		if result := c.ParseNtpq(out.NtpqPeers); result.NTPActive {
			return result
		}
	}

	for _, line := range strings.Split(out.SystemctlActive, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && parts[1] == "active" {
			daemon := strings.TrimSuffix(parts[0], ".service")
			return models.TimeSyncResult{
				NTPActive: true,
				NTPDaemon: daemon,
				NTPSynced: true,
			}
		}
	}

	return models.TimeSyncResult{
		NTPActive: false,
		Error:     "NTP daemon not found (timedatectl, chronyc, ntpq, systemctl all unavailable)",
	}
}

// ParseTimedatectlTimesync parses `timedatectl timesync-status` output.
// This is the most accurate source for cross-host drift measurement because
// it includes the Offset: field.
func (c *Checker) ParseTimedatectlTimesync(stdout string) models.TimeSyncResult {
	result := models.TimeSyncResult{
		NTPDaemon: "systemd-timesyncd",
		NTPActive: true,
	}
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Leap:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "Leap:"))
			result.NTPSynced = strings.EqualFold(val, "normal")
		}
		if strings.HasPrefix(line, "Offset:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "Offset:"))
			re := regexp.MustCompile(`([+-]?\d+\.?\d*)\s*(ms|us|µs|s)`)
			matches := re.FindStringSubmatch(val)
			if len(matches) >= 3 {
				num, _ := strconv.ParseFloat(matches[1], 64)
				unit := matches[2]
				switch unit {
				case "ms":
					result.ClockOffsetMs = int64(num)
				case "us", "µs":
					result.ClockOffsetMs = int64(num / 1000)
				case "s":
					result.ClockOffsetMs = int64(num * 1000)
				}
			}
		}
	}
	if !result.NTPSynced {
		result.Error = "systemd-timesyncd running but not synchronized (leap not normal)"
	}
	return result
}

func (c *Checker) ParseTimedatectl(stdout string) models.TimeSyncResult {
	result := models.TimeSyncResult{}
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "NTPService=") {
			val := strings.TrimPrefix(line, "NTPService=")
			if strings.Contains(val, "chronyd") {
				result.NTPDaemon = "chrony"
			} else if strings.Contains(val, "ntpd") || strings.Contains(val, "ntp") {
				result.NTPDaemon = "ntpd"
			}
		}
		if strings.HasPrefix(line, "NTPSynchronized=") {
			val := strings.TrimPrefix(line, "NTPSynchronized=")
			result.NTPSynced = strings.EqualFold(val, "yes")
		}
	}
	result.NTPActive = result.NTPDaemon != ""
	if result.NTPActive && !result.NTPSynced {
		result.Error = fmt.Sprintf("%s running but not synchronized", result.NTPDaemon)
	}
	return result
}

// ParseTimedatectlStatus parses plain `timedatectl` (status mode) output.
// Uses a two-pass approach because fields can appear in any order.
func (c *Checker) ParseTimedatectlStatus(stdout string) models.TimeSyncResult {
	result := models.TimeSyncResult{}
	var synced bool
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "NTP service:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "NTP service:"))
			if strings.EqualFold(val, "active") {
				result.NTPActive = true
				result.NTPDaemon = "systemd-timesyncd"
			}
		}
		if strings.HasPrefix(line, "System clock synchronized:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "System clock synchronized:"))
			synced = strings.EqualFold(val, "yes")
		}
	}
	result.NTPSynced = result.NTPActive && synced
	if result.NTPActive && !result.NTPSynced {
		result.Error = "systemd-timesyncd running but not synchronized"
	}
	return result
}

func (c *Checker) ParseChronyc(stdout string) models.TimeSyncResult {
	result := models.TimeSyncResult{
		NTPDaemon: "chrony",
		NTPActive: true,
	}
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Leap status") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "Leap status"))
			// Remove leading colon if present
			val = strings.TrimPrefix(val, ":")
			val = strings.TrimSpace(val)
			result.NTPSynced = strings.EqualFold(val, "Normal")
		}
		if strings.HasPrefix(line, "System time") {
			re := regexp.MustCompile(`(-?\d+\.?\d*)\s+seconds`)
			matches := re.FindStringSubmatch(line)
			if len(matches) >= 2 {
				offsetSec, _ := strconv.ParseFloat(matches[1], 64)
				result.ClockOffsetMs = int64(offsetSec * 1000)
			}
		}
	}
	if !result.NTPSynced {
		result.Error = "chrony running but not synchronized (leap status not Normal)"
	}
	return result
}

// ParseNtpq parses `ntpq -p` output. A line starting with '*' indicates the
// system peer (synchronized).
func (c *Checker) ParseNtpq(stdout string) models.TimeSyncResult {
	result := models.TimeSyncResult{
		NTPDaemon: "ntpd",
		NTPActive: true,
	}
	for _, line := range strings.Split(stdout, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "*") {
			result.NTPSynced = true
			fields := strings.Fields(line)
			if len(fields) >= 9 {
				offsetMs, _ := strconv.ParseFloat(fields[8], 64)
				result.ClockOffsetMs = int64(offsetMs)
			}
			break
		}
	}
	if !result.NTPSynced {
		result.Error = "ntpd running but not synchronized (no system peer)"
	}
	return result
}

// checkPortsAvailability checks the given ports against previously-collected
// ss(8)/netstat(8) output. If the output is empty (both tools unavailable on
// the remote host), all ports are treated as available since their state
// could not be determined.
func checkPortsAvailability(ssOutput string, ports []int) (details []models.PortCheckResult, allFree bool, errors []string) {
	if strings.TrimSpace(ssOutput) == "" {
		for _, port := range ports {
			details = append(details, models.PortCheckResult{Port: port, Available: true})
		}
		return details, true, nil
	}

	fmt.Printf("[DEBUG] checkPortsAvailability stdout=\n%s\n---END---\n", ssOutput)

	listenPorts := make(map[string]struct{})
	for _, line := range strings.Split(ssOutput, "\n") {
		if !strings.Contains(strings.ToUpper(line), "LISTEN") {
			continue
		}
		re := regexp.MustCompile(`:(\d+)`)
		for _, m := range re.FindAllStringSubmatch(line, -1) {
			listenPorts[m[1]] = struct{}{}
		}
	}

	allFree = true
	for _, port := range ports {
		portStr := strconv.Itoa(port)
		pr := models.PortCheckResult{Port: port}
		if _, taken := listenPorts[portStr]; taken {
			pr.Available = false
			pr.Error = fmt.Sprintf("port %s/tcp is already in use", portStr)
			errors = append(errors, pr.Error)
			allFree = false
		} else {
			pr.Available = true
		}
		details = append(details, pr)
	}

	return details, allFree, errors
}

func (c *Checker) cleanSSHError(output string, err error) string {
	output = strings.TrimSpace(output)
	if output != "" {
		lines := strings.Split(output, "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(lines[i])
			if line != "" {
				return line
			}
		}
	}
	if err != nil {
		return err.Error()
	}
	return "unknown error"
}
