package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

const (
	defaultSSHPort      = 22
	defaultSSHTimeout   = 10 * time.Second
	healthCheckInterval = 500 * time.Millisecond
)

type SSHConfig struct {
	Host           string
	User           string
	KeyPath        string
	KnownHostsPath string
	Timeout        time.Duration
}

type ServiceConfig struct {
	Namespace string
	Name      string
	Port      int
}

type Connector struct {
	sshClient *ssh.Client
	tunnels   []*tunnel
	mu        sync.Mutex
}

type ServiceTunnel struct {
	LocalAddr string
}

type tunnel struct {
	listener   net.Listener
	session    *ssh.Session
	localPort  int
	remotePort int
	done       chan struct{}
	closeOnce  sync.Once
}

func Connect(cfg SSHConfig) (*Connector, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultSSHTimeout
	}

	sshClient, err := dialSSH(cfg)
	if err != nil {
		return nil, fmt.Errorf("ssh connect: %w", err)
	}

	return &Connector{
		sshClient: sshClient,
		tunnels:   make([]*tunnel, 0),
	}, nil
}

func (c *Connector) Forward(ctx context.Context, svc ServiceConfig) (*ServiceTunnel, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	session, err := c.sshClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	cmd := fmt.Sprintf(
		"exec kubectl port-forward %s -n %s %d:%d",
		svc.Name, svc.Namespace, svc.Port, svc.Port,
	)

	if err := session.Start(cmd); err != nil {
		session.Close()
		return nil, fmt.Errorf("start port-forward: %w", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		session.Signal(ssh.SIGTERM)
		session.Close()
		return nil, fmt.Errorf("listen: %w", err)
	}

	localPort := listener.Addr().(*net.TCPAddr).Port
	done := make(chan struct{})

	t := &tunnel{
		listener:   listener,
		session:    session,
		localPort:  localPort,
		remotePort: svc.Port,
		done:       done,
	}

	go c.runTunnel(t)

	c.tunnels = append(c.tunnels, t)

	localAddr := fmt.Sprintf("127.0.0.1:%d", localPort)
	if err := waitForTunnel(ctx, c.sshClient, localAddr, svc.Port); err != nil {
		closeErr := c.closeTunnel(t)
		c.tunnels = c.tunnels[:len(c.tunnels)-1]
		if closeErr != nil {
			return nil, fmt.Errorf("service not reachable: %w (cleanup: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("service not reachable: %w", err)
	}

	return &ServiceTunnel{LocalAddr: localAddr}, nil
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
	c.mu.Lock()
	defer c.mu.Unlock()

	var errs []error

	for _, t := range c.tunnels {
		if err := c.closeTunnel(t); err != nil {
			errs = append(errs, err)
		}
	}
	c.tunnels = nil

	if c.sshClient != nil {
		if err := c.sshClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close ssh: %w", err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (c *Connector) closeTunnel(t *tunnel) error {
	var closeErr error

	t.closeOnce.Do(func() {
		var errs []error

		close(t.done)

		if t.listener != nil {
			if err := t.listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
				errs = append(errs, fmt.Errorf("close listener: %w", err))
			}
			t.listener = nil
		}

		if t.session != nil {
			_ = t.session.Signal(ssh.SIGTERM)
			if err := t.session.Close(); err != nil && !errors.Is(err, io.EOF) {
				errs = append(errs, fmt.Errorf("close session: %w", err))
			}
			t.session = nil
		}

		if len(errs) > 0 {
			closeErr = errors.Join(errs...)
		}
	})

	return closeErr
}

func (c *Connector) runTunnel(t *tunnel) {
	for {
		select {
		case <-t.done:
			return
		default:
		}

		localConn, err := t.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			continue
		}

		go c.handleTunnelConn(localConn, t.remotePort)
	}
}

func (c *Connector) handleTunnelConn(localConn net.Conn, remotePort int) {
	defer localConn.Close()

	remoteAddr := fmt.Sprintf("127.0.0.1:%d", remotePort)
	remoteConn, err := c.sshClient.Dial("tcp", remoteAddr)
	if err != nil {
		return
	}
	defer remoteConn.Close()

	done := make(chan struct{}, 2)

	go func() {
		io.Copy(remoteConn, localConn)
		done <- struct{}{}
	}()

	go func() {
		io.Copy(localConn, remoteConn)
		done <- struct{}{}
	}()

	<-done
}

func waitForTunnel(ctx context.Context, sshClient *ssh.Client, localAddr string, remotePort int) error {
	dialer := &net.Dialer{Timeout: 2 * time.Second}
	remoteAddr := fmt.Sprintf("127.0.0.1:%d", remotePort)
	var lastErr error

	for {
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("%w (last check: %v)", ctx.Err(), lastErr)
			}
			return ctx.Err()
		default:
		}

		localConn, err := dialer.DialContext(ctx, "tcp", localAddr)
		if err != nil {
			lastErr = fmt.Errorf("dial local %s: %w", localAddr, err)
		} else {
			localConn.Close()

			// Ensure the SSH-side endpoint is also reachable so we don't report
			// readiness while the remote port-forward is still starting.
			remoteConn, err := sshClient.Dial("tcp", remoteAddr)
			if err == nil {
				remoteConn.Close()
				return nil
			}
			lastErr = fmt.Errorf("dial remote %s: %w", remoteAddr, err)
		}

		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("%w (last check: %v)", ctx.Err(), lastErr)
			}
			return ctx.Err()
		case <-time.After(healthCheckInterval):
		}
	}
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
		// IPv6 addresses need brackets
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
