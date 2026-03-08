package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

const (
	defaultSSHPort    = 22
	defaultSSHTimeout = 10 * time.Second
)

type SSHConfig struct {
	Host           string
	User           string
	KeyPath        string
	KnownHostsPath string
	Timeout        time.Duration
}

type Connector struct {
	sshClient *ssh.Client
}

func Connect(cfg SSHConfig) (*Connector, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultSSHTimeout
	}

	sshClient, err := dialSSH(cfg)
	if err != nil {
		return nil, fmt.Errorf("ssh connect: %w", err)
	}

	return &Connector{sshClient: sshClient}, nil
}

func (c *Connector) RunCommand(cmd string) ([]byte, error) {
	return c.RunCommandContext(context.Background(), cmd)
}

func (c *Connector) RunCommandContext(ctx context.Context, cmd string) ([]byte, error) {
	session, err := c.sshClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	defer session.Close()

	type commandResult struct {
		output []byte
		err    error
	}

	resultCh := make(chan commandResult, 1)
	go func() {
		output, err := session.CombinedOutput(cmd)
		resultCh <- commandResult{output: output, err: err}
	}()

	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGTERM)
		return nil, fmt.Errorf("run command canceled: %w", ctx.Err())
	case result := <-resultCh:
		if result.err != nil {
			return nil, fmt.Errorf("run command: %w (output: %s)", result.err, result.output)
		}
		return result.output, nil
	}
}

func (c *Connector) Close() error {
	if c.sshClient != nil {
		if err := c.sshClient.Close(); err != nil {
			return fmt.Errorf("close ssh: %w", err)
		}
	}
	return nil
}

type HostInfo struct {
	IPv6Address string `json:"ipv6_address"`
}

func LoadHost(hostsFile, hostname string) (string, error) {
	data, err := os.ReadFile(hostsFile)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	var hosts map[string]HostInfo
	if err := json.Unmarshal(data, &hosts); err != nil {
		return "", fmt.Errorf("parse JSON: %w", err)
	}

	hostInfo, ok := hosts[hostname]
	if !ok {
		available := make([]string, 0, len(hosts))
		for name := range hosts {
			available = append(available, name)
		}
		return "", fmt.Errorf("host %q not found (available: %s)", hostname, strings.Join(available, ", "))
	}

	if hostInfo.IPv6Address == "" {
		return "", fmt.Errorf("host %q has no ipv6_address", hostname)
	}

	return hostInfo.IPv6Address, nil
}

func dialSSH(cfg SSHConfig) (*ssh.Client, error) {
	keyData, err := os.ReadFile(cfg.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("read key %s: %w", cfg.KeyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("parse key: %w", err)
	}

	hostKeyCallback, err := buildHostKeyCallback(cfg.KnownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("build host key callback: %w", err)
	}

	config := &ssh.ClientConfig{
		User: cfg.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: hostKeyCallback,
		Timeout:         cfg.Timeout,
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, defaultSSHPort)
	if strings.Contains(cfg.Host, ":") {
		addr = fmt.Sprintf("[%s]:%d", cfg.Host, defaultSSHPort)
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}

	return client, nil
}

func buildHostKeyCallback(path string) (ssh.HostKeyCallback, error) {
	if path == "" {
		return nil, fmt.Errorf("known_hosts path is required")
	}

	callback, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("load known_hosts file %s: %w", path, err)
	}

	return callback, nil
}
