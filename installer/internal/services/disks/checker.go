package disks

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"ydb-installer/internal/models"
	"ydb-installer/internal/services/sshclient"
)

type Checker struct{}

func NewChecker() *Checker {
	return &Checker{}
}

func (c *Checker) CheckDisks(req *models.CheckDisksRequest) (*models.GetDisksResponse, error) {
	if len(req.Hosts) == 0 {
		return &models.GetDisksResponse{Results: []models.HostDisksResult{}}, nil
	}

	if req.SSH != nil {
		if err := sshclient.ValidateKeyPath(req.SSH.KeyPath); err != nil {
			return nil, err
		}
		if req.SSH.UseJumpHost && req.SSH.JumpHostFQDN == "" {
			return nil, fmt.Errorf("Jump host FQDN is required when use_jump_host is true")
		}
	}

	results := make([]models.HostDisksResult, len(req.Hosts))
	var wg sync.WaitGroup

	for i, host := range req.Hosts {
		wg.Add(1)
		go func(idx int, h string) {
			defer wg.Done()
			results[idx] = c.discoverDisks(h, req.SSH)
		}(i, host)
	}

	wg.Wait()

	return &models.GetDisksResponse{Results: results}, nil
}

func (c *Checker) CheckDiskConsistency(results []models.HostDisksResult) *models.DiskConsistencyResult {
	consistency := &models.DiskConsistencyResult{
		IsConsistent: true,
		Issues:       []models.DiskConsistencyIssue{},
		CommonDisks:  []models.DiskInfo{},
	}

	if len(results) == 0 {
		return consistency
	}

	hostDiskCounts := make(map[string]int)
	hostDiskTypes := make(map[string][]string)
	hostDiskSizes := make(map[string][]int64)

	for _, result := range results {
		if result.Error != "" {
			continue
		}
		hostDiskCounts[result.Host] = len(result.Disks)

		types := make([]string, 0, len(result.Disks))
		sizes := make([]int64, 0, len(result.Disks))
		for _, disk := range result.Disks {
			types = append(types, disk.Type)
			sizes = append(sizes, disk.SizeBytes)
		}
		hostDiskTypes[result.Host] = types
		hostDiskSizes[result.Host] = sizes
	}

	if len(hostDiskCounts) > 1 {
		counts := make(map[int][]string)
		for host, count := range hostDiskCounts {
			counts[count] = append(counts[count], host)
		}
		if len(counts) > 1 {
			consistency.IsConsistent = false
			for count, hosts := range counts {
				consistency.Issues = append(consistency.Issues, models.DiskConsistencyIssue{
					Type:    "count_mismatch",
					Hosts:   hosts,
					Message: fmt.Sprintf("Hosts %v have %d disks (expected same count on all hosts)", hosts, count),
				})
			}
		}
	}

	if len(hostDiskTypes) > 1 {
		typeSets := make(map[string][]string)
		for host, types := range hostDiskTypes {
			typeKey := strings.Join(types, ",")
			typeSets[typeKey] = append(typeSets[typeKey], host)
		}
		if len(typeSets) > 1 {
			consistency.IsConsistent = false
			for typeKey, hosts := range typeSets {
				consistency.Issues = append(consistency.Issues, models.DiskConsistencyIssue{
					Type:    "type_mismatch",
					Hosts:   hosts,
					Message: fmt.Sprintf("Hosts %v have disk types: %s", hosts, typeKey),
				})
			}
		}
	}

	if len(hostDiskSizes) > 1 {
		sizeSets := make(map[string][]string)
		for host, sizes := range hostDiskSizes {
			sizeKey := fmt.Sprintf("%v", sizes)
			sizeSets[sizeKey] = append(sizeSets[sizeKey], host)
		}
		if len(sizeSets) > 1 {
			consistency.IsConsistent = false
			for sizeKey, hosts := range sizeSets {
				consistency.Issues = append(consistency.Issues, models.DiskConsistencyIssue{
					Type:    "size_mismatch",
					Hosts:   hosts,
					Message: fmt.Sprintf("Hosts %v have disk sizes: %s", hosts, sizeKey),
				})
			}
		}
	}

	if len(results) > 0 {
		refHost := results[0]
		if refHost.Error == "" && len(refHost.Disks) > 0 {
			allMatch := true
			for _, result := range results[1:] {
				if result.Error != "" || len(result.Disks) != len(refHost.Disks) {
					allMatch = false
					break
				}
				for i, disk := range result.Disks {
					if disk.Name != refHost.Disks[i].Name ||
						disk.Type != refHost.Disks[i].Type ||
						disk.SizeBytes != refHost.Disks[i].SizeBytes {
						allMatch = false
						break
					}
				}
			}
			if allMatch {
				consistency.CommonDisks = refHost.Disks
			}
		}
	}

	return consistency
}

func (c *Checker) discoverDisks(host string, ssh *models.SSHConfig) models.HostDisksResult {
	result := models.HostDisksResult{
		Host:  host,
		Disks: []models.DiskInfo{},
	}

	sshArgs := sshclient.BuildSSHArgs(host, ssh)

	runSSH := func(extraArgs []string, timeout time.Duration) (string, string, error) {
		cmdArgs := append(append([]string{}, sshArgs...), extraArgs...)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		cmd := exec.CommandContext(ctx, "ssh", cmdArgs...)
		cmd.Env = append(os.Environ(), "LANG=C")
		var stdoutBuf, stderrBuf bytes.Buffer
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf
		err := cmd.Run()
		return stdoutBuf.String(), stderrBuf.String(), err
	}

	// Retry once with 1s backoff to absorb transient network blips.
	const maxAttempts = 2
	var (
		stdout, stderr string
		sshErr         error
	)
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		stdout, stderr, sshErr = runSSH([]string{
			"lsblk", "-b", "-n", "-o", "NAME,TYPE,SIZE,ROTA,FSTYPE,MOUNTPOINT,PKNAME", "--noheadings",
		}, 20*time.Second)
		if sshErr == nil {
			break
		}
		if attempt < maxAttempts {
			time.Sleep(1 * time.Second)
		}
	}

	if sshErr != nil {
		errMsg := stderr
		if errMsg == "" {
			errMsg = sshErr.Error()
		}
		result.Error = fmt.Sprintf("Failed to get disk list: %s", c.cleanSSHError(errMsg, sshErr))
		return result
	}

	var allDevices []lsblkDevice
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		device := c.parseLsblkLine(line)
		if device != nil {
			allDevices = append(allDevices, *device)
		}
	}

	// A disk is a system disk if any of its partitions are mounted at critical paths.
	criticalMountPoints := map[string]bool{
		"/":     true,
		"/boot": true,
		"/home": true,
		"/var":  true,
		"/usr":  true,
		"/opt":  true,
		"/srv":  true,
		"/tmp":  true,
		"/root": true,
	}

	diskHasSystemPartition := make(map[string]bool)

	for _, device := range allDevices {
		if device.Type == "part" && device.MountPoint != "" {
			if criticalMountPoints[device.MountPoint] {
				if device.ParentDisk != "" {
					diskHasSystemPartition[device.ParentDisk] = true
				}
			}
		}
	}

	for _, device := range allDevices {
		if device.Type == "disk" {
			diskInfo := models.DiskInfo{
				Name:         device.Name,
				Path:         "/dev/" + device.Name,
				Type:         device.DiskType,
				Size:         c.formatBytes(device.SizeBytes),
				SizeBytes:    device.SizeBytes,
				Mounted:      device.Mounted,
				MountPoint:   device.MountPoint,
				Filesystem:   device.Filesystem,
				IsSystemDisk: diskHasSystemPartition[device.Name],
			}
			result.Disks = append(result.Disks, diskInfo)
		}
	}

	return result
}

type lsblkDevice struct {
	Name       string
	Type       string
	SizeBytes  int64
	DiskType   string
	Mounted    bool
	MountPoint string
	Filesystem string
	ParentDisk string
}

func (c *Checker) parseLsblkLine(line string) *lsblkDevice {
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return nil
	}

	name := fields[0]
	devType := fields[1]
	sizeStr := fields[2]
	rota := fields[3]

	if devType != "disk" && devType != "part" {
		return nil
	}

	sizeBytes, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		sizeBytes = 0
	}

	diskType := "HDD"
	if devType == "disk" {
		if strings.HasPrefix(name, "nvme") {
			diskType = "NVMe"
		} else if rota == "0" {
			diskType = "SSD"
		}
	}

	filesystem := ""
	mountPoint := ""
	if len(fields) >= 5 {
		filesystem = fields[4]
	}
	if len(fields) >= 6 {
		mountPoint = fields[5]
	}

	parentDisk := ""
	if len(fields) >= 7 {
		parentDisk = fields[6]
	}

	mounted := mountPoint != "" || filesystem != ""

	return &lsblkDevice{
		Name:       name,
		Type:       devType,
		SizeBytes:  sizeBytes,
		DiskType:   diskType,
		Mounted:    mounted,
		MountPoint: mountPoint,
		Filesystem: filesystem,
		ParentDisk: parentDisk,
	}
}

func (c *Checker) formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.1fT", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.1fG", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1fM", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1fK", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
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
	return err.Error()
}

func (c *Checker) ValidateDisks(req *models.ValidateDisksRequest) (*models.ValidateDisksResponse, error) {
	if len(req.Hosts) == 0 {
		return &models.ValidateDisksResponse{Results: []models.DiskValidationResult{}}, nil
	}

	if len(req.Paths) == 0 {
		return &models.ValidateDisksResponse{Results: []models.DiskValidationResult{}}, nil
	}

	if req.SSH != nil {
		if err := sshclient.ValidateKeyPath(req.SSH.KeyPath); err != nil {
			return nil, err
		}
		if req.SSH.UseJumpHost && req.SSH.JumpHostFQDN == "" {
			return nil, fmt.Errorf("Jump host FQDN is required when use_jump_host is true")
		}
	}

	pathSet := make(map[string]bool, len(req.Paths))
	for _, p := range req.Paths {
		pathSet[p] = true
	}

	results := make([]models.DiskValidationResult, len(req.Hosts))
	var wg sync.WaitGroup

	for i, host := range req.Hosts {
		wg.Add(1)
		go func(idx int, h string) {
			defer wg.Done()
			results[idx] = c.validateDisksOnHost(h, req.SSH, pathSet)
		}(i, host)
	}

	wg.Wait()

	return &models.ValidateDisksResponse{Results: results}, nil
}

func (c *Checker) validateDisksOnHost(host string, ssh *models.SSHConfig, paths map[string]bool) models.DiskValidationResult {
	result := models.DiskValidationResult{
		Host:  host,
		Disks: []models.DiskValidation{},
	}

	sshArgs := sshclient.BuildSSHArgs(host, ssh)

	runSSH := func(extraArgs []string, timeout time.Duration) (string, string, error) {
		cmdArgs := append(append([]string{}, sshArgs...), extraArgs...)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		cmd := exec.CommandContext(ctx, "ssh", cmdArgs...)
		cmd.Env = append(os.Environ(), "LANG=C")
		var stdoutBuf, stderrBuf bytes.Buffer
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf
		err := cmd.Run()
		return stdoutBuf.String(), stderrBuf.String(), err
	}

	const maxAttempts = 2
	var (
		stdout, stderr string
		sshErr         error
	)
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		stdout, stderr, sshErr = runSSH([]string{
			"lsblk", "-b", "-n", "-o", "NAME,PATH", "--noheadings",
		}, 20*time.Second)
		if sshErr == nil {
			break
		}
		if attempt < maxAttempts {
			time.Sleep(1 * time.Second)
		}
	}

	if sshErr != nil {
		errMsg := stderr
		if errMsg == "" {
			errMsg = sshErr.Error()
		}
		result.Error = fmt.Sprintf("SSH error: %s", c.cleanSSHError(errMsg, sshErr))
		for path := range paths {
			result.Disks = append(result.Disks, models.DiskValidation{
				Path:  path,
				Found: false,
			})
		}
		return result
	}

	foundPaths := make(map[string]bool)
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		foundPaths[fields[1]] = true
	}

	result.Valid = true
	for path := range paths {
		found := foundPaths[path]
		result.Disks = append(result.Disks, models.DiskValidation{
			Path:  path,
			Found: found,
		})
		if !found {
			result.Valid = false
		}
	}

	return result
}
