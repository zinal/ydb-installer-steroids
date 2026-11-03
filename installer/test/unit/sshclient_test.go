package unit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ydb-installer/internal/models"
	"ydb-installer/internal/services/sshclient"
)

func TestBuildSSHArgs_BasicHost(t *testing.T) {
	args := sshclient.BuildSSHArgs("example.com", nil)

	if len(args) == 0 {
		t.Fatal("Expected SSH args, got empty slice")
	}

	argStr := strings.Join(args, " ")

	if !strings.Contains(argStr, "ConnectTimeout=10") {
		t.Error("Expected ConnectTimeout=10 in args")
	}
	if !strings.Contains(argStr, "ServerAliveInterval=5") {
		t.Error("Expected ServerAliveInterval=5 in args")
	}
	if !strings.Contains(argStr, "ServerAliveCountMax=2") {
		t.Error("Expected ServerAliveCountMax=2 in args")
	}
	if !strings.Contains(argStr, "StrictHostKeyChecking=no") {
		t.Error("Expected StrictHostKeyChecking=no in args")
	}
	if !strings.Contains(argStr, "UserKnownHostsFile=/dev/null") {
		t.Error("Expected UserKnownHostsFile=/dev/null in args (must be paired with StrictHostKeyChecking=no so changed host keys never block a connection)")
	}
	if !strings.Contains(argStr, "BatchMode=yes") {
		t.Error("Expected BatchMode=yes in args")
	}

	if args[len(args)-1] != "example.com" {
		t.Errorf("Expected last arg to be 'example.com', got '%s'", args[len(args)-1])
	}
}

func TestBuildSSHArgs_WithLogin(t *testing.T) {
	ssh := &models.SSHConfig{Login: "admin"}
	args := sshclient.BuildSSHArgs("example.com", ssh)

	argStr := strings.Join(args, " ")
	if !strings.Contains(argStr, "admin@example.com") {
		t.Error("Expected admin@example.com in args")
	}
}

func TestBuildSSHArgs_WithKeyAuth(t *testing.T) {
	ssh := &models.SSHConfig{
		AuthType: "key",
		KeyPath:  "/home/user/.ssh/id_rsa",
	}
	args := sshclient.BuildSSHArgs("example.com", ssh)

	argStr := strings.Join(args, " ")
	if !strings.Contains(argStr, "-i") {
		t.Error("Expected -i flag in args")
	}
	if !strings.Contains(argStr, "/home/user/.ssh/id_rsa") {
		t.Error("Expected key path in args")
	}
	if !strings.Contains(argStr, "PasswordAuthentication=no") {
		t.Error("Expected PasswordAuthentication=no in args")
	}
	if !strings.Contains(argStr, "PreferredAuthentications=publickey") {
		t.Error("Expected PreferredAuthentications=publickey in args")
	}
}

func TestBuildSSHArgs_WithJumpHost(t *testing.T) {
	ssh := &models.SSHConfig{
		UseJumpHost:  true,
		JumpHostFQDN: "jump.example.com",
		Login:        "root",
		AuthType:     "key",
		KeyPath:      "/home/user/.ssh/id_rsa",
	}
	args := sshclient.BuildSSHArgs("target.example.com", ssh)

	argStr := strings.Join(args, " ")

	if !strings.Contains(argStr, "ProxyCommand=") {
		t.Error("Expected ProxyCommand in args")
	}

	if !strings.Contains(argStr, "root@jump.example.com") {
		t.Error("Expected jump host with login in args")
	}

	if !strings.Contains(argStr, "ConnectTimeout=10") {
		t.Error("Expected ConnectTimeout in ProxyCommand args")
	}

	// Check ProxyCommand also disables host key verification, otherwise a
	// changed jump host key would block the connection even with the outer
	// StrictHostKeyChecking=no set.
	if strings.Count(argStr, "UserKnownHostsFile=/dev/null") < 2 {
		t.Error("Expected UserKnownHostsFile=/dev/null in both the main SSH args and the ProxyCommand")
	}

	if args[len(args)-1] != "root@target.example.com" {
		t.Errorf("Expected last arg to be 'root@target.example.com', got '%s'", args[len(args)-1])
	}
}

func TestBuildSSHArgs_WithJumpHostNoLogin(t *testing.T) {
	ssh := &models.SSHConfig{
		UseJumpHost:  true,
		JumpHostFQDN: "jump.example.com",
	}
	args := sshclient.BuildSSHArgs("target.example.com", ssh)

	argStr := strings.Join(args, " ")

	if !strings.Contains(argStr, "jump.example.com") {
		t.Error("Expected jump host in args")
	}

	if args[len(args)-1] != "target.example.com" {
		t.Errorf("Expected last arg to be 'target.example.com', got '%s'", args[len(args)-1])
	}
}

func TestBuildSSHArgs_WithKeyPathContainingSpaces(t *testing.T) {
	ssh := &models.SSHConfig{
		UseJumpHost:  true,
		JumpHostFQDN: "jump.example.com",
		AuthType:     "key",
		KeyPath:      "/home/user/my key",
	}
	args := sshclient.BuildSSHArgs("target.example.com", ssh)

	argStr := strings.Join(args, " ")

	if !strings.Contains(argStr, "'/home/user/my key'") {
		t.Error("Expected quoted key path in ProxyCommand")
	}
}

func TestBuildSSHArgs_WithCustomSSHArgs(t *testing.T) {
	ssh := &models.SSHConfig{
		CustomSSHArgs: "-o LogLevel=ERROR -o Compression=yes",
	}
	args := sshclient.BuildSSHArgs("example.com", ssh)

	argStr := strings.Join(args, " ")
	if !strings.Contains(argStr, "LogLevel=ERROR") {
		t.Error("Expected LogLevel=ERROR in args")
	}
	if !strings.Contains(argStr, "Compression=yes") {
		t.Error("Expected Compression=yes in args")
	}
}

func TestParseShellArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple args",
			input:    "-o StrictHostKeyChecking=no -o BatchMode=yes",
			expected: []string{"-o", "StrictHostKeyChecking=no", "-o", "BatchMode=yes"},
		},
		{
			name:     "single quoted string",
			input:    "-i '/path/with spaces/key'",
			expected: []string{"-i", "/path/with spaces/key"},
		},
		{
			name:     "double quoted string",
			input:    `-i "/path/with spaces/key"`,
			expected: []string{"-i", "/path/with spaces/key"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "multiple spaces",
			input:    "-o  foo   -o  bar",
			expected: []string{"-o", "foo", "-o", "bar"},
		},
		{
			name:     "escaped characters",
			input:    `foo\ bar`,
			expected: []string{"foo bar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sshclient.ParseShellArgs(tt.input)
			if err != nil {
				t.Errorf("ParseShellArgs() error = %v", err)
			}
			if len(result) != len(tt.expected) {
				t.Errorf("ParseShellArgs() len = %d, want %d", len(result), len(tt.expected))
			}
			for i, arg := range result {
				if i < len(tt.expected) && arg != tt.expected[i] {
					t.Errorf("ParseShellArgs()[%d] = %q, want %q", i, arg, tt.expected[i])
				}
			}
		})
	}
}

func TestBuildSSHArgs_TimeoutOptionsInProxyCommand(t *testing.T) {
	// This is the critical test — ensures that the ProxyCommand SSH also has
	// timeout options, so it won't hang when the jump host is unreachable
	ssh := &models.SSHConfig{
		UseJumpHost:  true,
		JumpHostFQDN: "unreachable-jump.example.com",
		Login:        "root",
	}
	args := sshclient.BuildSSHArgs("target.example.com", ssh)

	argStr := strings.Join(args, " ")

	// Count how many times ConnectTimeout appears — should be at least 2
	// (once for main SSH, once for ProxyCommand SSH)
	count := strings.Count(argStr, "ConnectTimeout=10")
	if count < 2 {
		t.Errorf("Expected ConnectTimeout=10 to appear at least 2 times (main + proxy), got %d", count)
	}

	if strings.Count(argStr, "ServerAliveInterval=5") < 2 {
		t.Error("Expected ServerAliveInterval=5 in both main SSH and ProxyCommand")
	}
	if strings.Count(argStr, "ServerAliveCountMax=2") < 2 {
		t.Error("Expected ServerAliveCountMax=2 in both main SSH and ProxyCommand")
	}
}

func TestBuildSSHArgs_ExpandsTildeInKeyPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("home directory unavailable")
	}

	ssh := &models.SSHConfig{
		AuthType: "key",
		KeyPath:  "~/.ssh/id_rsa",
	}
	args := sshclient.BuildSSHArgs("example.com", ssh)

	expanded := filepath.Join(home, ".ssh/id_rsa")
	found := false
	for i, arg := range args {
		if arg == "-i" && i+1 < len(args) {
			if args[i+1] != expanded {
				t.Errorf("Expected -i %q, got -i %q", expanded, args[i+1])
			}
			if strings.HasPrefix(args[i+1], "~") {
				t.Errorf("Key path must be expanded, still starts with ~: %q", args[i+1])
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected -i flag with expanded key path")
	}
}

func TestBuildSSHArgs_ExpandsTildeInProxyCommandKeyPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("home directory unavailable")
	}

	ssh := &models.SSHConfig{
		UseJumpHost:  true,
		JumpHostFQDN: "jump.example.com",
		AuthType:     "key",
		KeyPath:      "~/.ssh/id_ed25519",
	}
	args := sshclient.BuildSSHArgs("target.example.com", ssh)

	expanded := filepath.Join(home, ".ssh/id_ed25519")
	argStr := strings.Join(args, " ")

	if !strings.Contains(argStr, expanded) {
		t.Errorf("Expected expanded key path %q in ProxyCommand args, got: %s", expanded, argStr)
	}
	// Literal tilde form must not appear as the -i value in ProxyCommand
	if strings.Contains(argStr, "-i ~/.ssh/id_ed25519") {
		t.Error("ProxyCommand still uses unexpanded tilde key path")
	}
}

func TestBuildSSHArgs_AbsoluteKeyPathUnchanged(t *testing.T) {
	abs := "/home/user/.ssh/custom_key"
	ssh := &models.SSHConfig{
		AuthType: "key",
		KeyPath:  abs,
	}
	args := sshclient.BuildSSHArgs("example.com", ssh)

	for i, arg := range args {
		if arg == "-i" && i+1 < len(args) {
			if args[i+1] != abs {
				t.Errorf("Absolute key path changed: got %q, want %q", args[i+1], abs)
			}
			return
		}
	}
	t.Error("Expected -i flag with absolute key path")
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("home directory unavailable")
	}

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "tilde slash", in: "~/.ssh/id_rsa", want: filepath.Join(home, ".ssh/id_rsa")},
		{name: "tilde only", in: "~", want: home},
		{name: "absolute", in: "/tmp/key", want: "/tmp/key"},
		{name: "relative", in: "relative/key", want: "relative/key"},
		{name: "empty", in: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sshclient.ExpandPath(tt.in)
			if got != tt.want {
				t.Errorf("ExpandPath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestValidateKeyPath(t *testing.T) {
	// Empty path is always OK (default key discovery)
	if err := sshclient.ValidateKeyPath(""); err != nil {
		t.Errorf("ValidateKeyPath(\"\") unexpected error: %v", err)
	}

	err := sshclient.ValidateKeyPath("/tmp/ydb-installer-missing-key-test-noexist")
	if err == nil {
		t.Error("ValidateKeyPath(missing absolute) expected error")
	} else if !strings.Contains(err.Error(), "SSH key file not found") {
		t.Errorf("unexpected error message: %v", err)
	}

	err = sshclient.ValidateKeyPath("~/ydb-installer-missing-key-test-noexist")
	if err == nil {
		t.Error("ValidateKeyPath(missing tilde) expected error")
	} else if !strings.Contains(err.Error(), "SSH key file not found") {
		t.Errorf("unexpected error message: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("home directory unavailable")
	}
	name := ".ydb-installer-validate-key-test"
	full := filepath.Join(home, name)
	if err := os.WriteFile(full, []byte("test-key"), 0o600); err != nil {
		t.Fatalf("failed to create temp key file: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(full) })

	if err := sshclient.ValidateKeyPath("~/" + name); err != nil {
		t.Errorf("ValidateKeyPath(existing tilde) unexpected error: %v", err)
	}
	if err := sshclient.ValidateKeyPath(full); err != nil {
		t.Errorf("ValidateKeyPath(existing absolute) unexpected error: %v", err)
	}
}
