package cluster

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	defaultSSHPort    = 22
	defaultSSHTimeout = 10 * time.Second
)

type SSHConfig struct {
	Host    string
	User    string
	KeyPath string
	Timeout time.Duration
}

type ServiceConfig struct {
	Namespace string
	Name      string
	Port      int
}

type Connector struct {
	LocalAddr      string
	sshClient      *ssh.Client
	portFwdSession *ssh.Session
	listener       net.Listener
}

func Connect(sshCfg SSHConfig, svcCfg ServiceConfig) (*Connector, error) {
	if sshCfg.Timeout == 0 {
		sshCfg.Timeout = defaultSSHTimeout
	}

	sshClient, err := dialSSH(sshCfg)
	if err != nil {
		return nil, fmt.Errorf("ssh connect: %w", err)
	}

	portFwdSession, err := startPortForward(sshClient, svcCfg)
	if err != nil {
		sshClient.Close()
		return nil, fmt.Errorf("kubectl port-forward: %w", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		portFwdSession.Signal(ssh.SIGTERM)
		portFwdSession.Close()
		sshClient.Close()
		return nil, fmt.Errorf("listen: %w", err)
	}

	localPort := listener.Addr().(*net.TCPAddr).Port
	go runTunnel(sshClient, listener, svcCfg.Port)

	return &Connector{
		LocalAddr:      fmt.Sprintf("127.0.0.1:%d", localPort),
		sshClient:      sshClient,
		portFwdSession: portFwdSession,
		listener:       listener,
	}, nil
}

func (c *Connector) RunCommand(cmd string) ([]byte, error) {
	session, err := c.sshClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return nil, fmt.Errorf("run command: %w (output: %s)", err, output)
	}

	return output, nil
}

func (c *Connector) Close() error {
	var errs []error

	if c.listener != nil {
		if err := c.listener.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close listener: %w", err))
		}
	}

	if c.portFwdSession != nil {
		_ = c.portFwdSession.Signal(ssh.SIGTERM)
		c.portFwdSession.Close()
	}

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

	config := &ssh.ClientConfig{
		User: cfg.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
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

func startPortForward(sshClient *ssh.Client, svc ServiceConfig) (*ssh.Session, error) {
	session, err := sshClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	cmd := fmt.Sprintf(
		"exec kubectl port-forward %s -n %s %d:%d",
		svc.Name, svc.Namespace, svc.Port, svc.Port,
	)

	if err := session.Start(cmd); err != nil {
		session.Close()
		return nil, fmt.Errorf("start: %w", err)
	}

	return session, nil
}

func runTunnel(sshClient *ssh.Client, listener net.Listener, remotePort int) {
	for {
		localConn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			continue
		}

		go handleTunnelConn(sshClient, localConn, remotePort)
	}
}

func handleTunnelConn(sshClient *ssh.Client, localConn net.Conn, remotePort int) {
	defer localConn.Close()

	remoteAddr := fmt.Sprintf("127.0.0.1:%d", remotePort)
	remoteConn, err := sshClient.Dial("tcp", remoteAddr)
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
