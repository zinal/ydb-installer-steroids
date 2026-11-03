// Package sshclient provides shared SSH argument building with proper
// timeout options to prevent hangs when jump hosts are unreachable.
package sshclient

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ydb-installer/internal/models"
)

const (
	ConnectTimeout      = "10" // 10s max for TCP connection
	ServerAliveInterval = "5"  // send keepalive every 5s
	ServerAliveCountMax = "2"  // give up after 2 missed keepalives (~10s)
)

// BuildSSHArgs constructs SSH command arguments from the SSH config.
// It includes proper timeout options to prevent hangs when jump hosts
// or target hosts are unreachable.
//
// UserKnownHostsFile=/dev/null is set together with StrictHostKeyChecking=no
// to completely disable host key verification. Without this, SSH refuses
// connections when a host key has changed (e.g., host reinstalled), even
// though StrictHostKeyChecking=no only auto-accepts *new* keys.
func BuildSSHArgs(host string, ssh *models.SSHConfig) []string {
	args := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=" + ConnectTimeout,
		"-o", "ServerAliveInterval=" + ServerAliveInterval,
		"-o", "ServerAliveCountMax=" + ServerAliveCountMax,
		// Connection multiplexing — reuse TCP connections for repeated
		// SSH commands to the same host, drastically reducing latency.
		// ControlPersist=30 keeps the master alive for 30s after the last
		// child exits, covering the entire parallel disk-check window.
		"-o", "ControlMaster=auto",
		"-o", "ControlPath=/tmp/ydb-installer-ssh-%C",
		"-o", "ControlPersist=30",
	}

	// Add SSH key if specified (expand ~/ so exec.Command does not depend on shell/OpenSSH tilde handling)
	if ssh != nil && ssh.AuthType == "key" && ssh.KeyPath != "" {
		keyPath := ExpandPath(ssh.KeyPath)
		args = append(args, "-o", "PasswordAuthentication=no")
		args = append(args, "-o", "PreferredAuthentications=publickey")
		args = append(args, "-i", keyPath)
	}

	if ssh != nil && ssh.UseJumpHost && ssh.JumpHostFQDN != "" {
		proxyCmd := buildProxyCommand(ssh)
		args = append(args, "-o", "ProxyCommand="+proxyCmd)
	}

	if ssh != nil && ssh.CustomSSHArgs != "" {
		parts, err := ParseShellArgs(ssh.CustomSSHArgs)
		if err != nil {
			fmt.Printf("Warning: Failed to parse custom SSH args: %v\n", err)
		}
		args = append(args, parts...)
	}

	target := host
	if ssh != nil && ssh.Login != "" {
		target = ssh.Login + "@" + host
	}
	args = append(args, target)

	return args
}

// buildProxyCommand constructs the ProxyCommand for jump host connections.
// The proxy command includes the same timeout options as the main SSH.
func buildProxyCommand(ssh *models.SSHConfig) string {
	jumpTarget := ssh.JumpHostFQDN
	if ssh.Login != "" {
		jumpTarget = ssh.Login + "@" + ssh.JumpHostFQDN
	}

	var parts []string
	parts = append(parts, "ssh")
	parts = append(parts, "-o", "StrictHostKeyChecking=no")
	parts = append(parts, "-o", "UserKnownHostsFile=/dev/null")
	parts = append(parts, "-o", "BatchMode=yes")
	parts = append(parts, "-o", "ConnectTimeout="+ConnectTimeout)
	parts = append(parts, "-o", "ServerAliveInterval="+ServerAliveInterval)
	parts = append(parts, "-o", "ServerAliveCountMax="+ServerAliveCountMax)

	// Quote the expanded key path if it contains spaces or special characters
	if ssh.AuthType == "key" && ssh.KeyPath != "" {
		keyPath := ExpandPath(ssh.KeyPath)
		if strings.ContainsAny(keyPath, " \t\n\r\"'`$\\&|;<>()[]{}*?") {
			parts = append(parts, "-i", "'"+keyPath+"'")
		} else {
			parts = append(parts, "-i", keyPath)
		}
	}

	parts = append(parts, "-W", "%h:%p", jumpTarget)
	return strings.Join(parts, " ")
}

// ExpandPath expands a leading "~/" to the current user's home directory.
// Returns the original path unchanged if it doesn't start with "~/".
func ExpandPath(path string) string {
	if !strings.HasPrefix(path, "~/") && path != "~" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}

// ValidateKeyPath checks that the SSH key file exists when a path is provided.
// Returns nil if path is empty (SSH will use default key discovery).
func ValidateKeyPath(keyPath string) error {
	if keyPath == "" {
		return nil
	}
	expanded := ExpandPath(keyPath)
	if _, err := os.Stat(expanded); os.IsNotExist(err) {
		return fmt.Errorf("SSH key file not found: %s", keyPath)
	}
	return nil
}

// ParseShellArgs parses shell arguments while preserving quoted strings.
// This is exported so it can be tested independently.
func ParseShellArgs(input string) ([]string, error) {
	var args []string
	var current strings.Builder
	var inSingleQuote bool
	var inDoubleQuote bool
	var escapeNext bool

	for _, ch := range input {
		if escapeNext {
			current.WriteRune(ch)
			escapeNext = false
			continue
		}

		switch ch {
		case '\\':
			escapeNext = true
		case '\'':
			if inDoubleQuote {
				current.WriteRune(ch)
			} else if inSingleQuote {
				inSingleQuote = false
			} else {
				inSingleQuote = true
			}
		case '"':
			if inSingleQuote {
				current.WriteRune(ch)
			} else if inDoubleQuote {
				inDoubleQuote = false
			} else {
				inDoubleQuote = true
			}
		case ' ', '\t':
			if inSingleQuote || inDoubleQuote {
				current.WriteRune(ch)
			} else if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args, nil
}
