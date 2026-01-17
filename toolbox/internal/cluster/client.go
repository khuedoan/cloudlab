package cluster

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/vault/api"
)

const (
	vaultNamespace = "vault"
	vaultService   = "svc/vault"
	vaultPort      = 8200
)

type ClientConfig struct {
	HostsFile string
	Host      string
	SSHUser   string
	SSHKey    string
	Timeout   time.Duration
}

type Client struct {
	conn  *Connector
	vault *api.Client
}

func NewClient(ctx context.Context, cfg ClientConfig) (*Client, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	hostAddr, err := LoadHost(cfg.HostsFile, cfg.Host)
	if err != nil {
		return nil, fmt.Errorf("load host: %w", err)
	}

	conn, err := Connect(SSHConfig{
		Host:    hostAddr,
		User:    cfg.SSHUser,
		KeyPath: cfg.SSHKey,
	})
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	token, err := getVaultToken(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("get vault token: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	vaultTunnel, err := conn.Forward(ctx, ServiceConfig{
		Namespace: vaultNamespace,
		Name:      vaultService,
		Port:      vaultPort,
	})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("forward vault: %w", err)
	}

	vaultClient, err := newVaultClient(vaultTunnel.LocalAddr, token)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("create vault client: %w", err)
	}

	return &Client{
		conn:  conn,
		vault: vaultClient,
	}, nil
}

func (c *Client) Vault() *api.Client {
	return c.vault
}

func (c *Client) Forward(ctx context.Context, svc ServiceConfig) (*ServiceTunnel, error) {
	return c.conn.Forward(ctx, svc)
}

func (c *Client) RunCommand(cmd string) ([]byte, error) {
	return c.conn.RunCommand(cmd)
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func getVaultToken(conn *Connector) (string, error) {
	cmd := fmt.Sprintf(
		`kubectl get secret vault-unseal-keys -n %s -o template='{{ index .data "vault-root" }}'`,
		vaultNamespace,
	)

	output, err := conn.RunCommand(cmd)
	if err != nil {
		return "", err
	}

	token, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(output)))
	if err != nil {
		return "", fmt.Errorf("decode token: %w", err)
	}

	return string(token), nil
}

func newVaultClient(addr, token string) (*api.Client, error) {
	config := api.DefaultConfig()
	config.Address = "http://" + addr

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}
	client.SetToken(token)

	return client, nil
}
