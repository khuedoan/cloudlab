package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/api"
)

const (
	vaultNamespace = "vault"
	vaultService   = "svc/vault"
	vaultPort      = 8200
)

func connectVault(ctx context.Context) (*api.Client, func(), error) {
	token, err := getLocalVaultToken(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("get vault token: %w", err)
	}

	forward, err := startKubectlPortForward(ctx, vaultNamespace, vaultService, vaultPort)
	if err != nil {
		return nil, nil, fmt.Errorf("forward vault: %w", err)
	}

	config := api.DefaultConfig()
	config.Address = "http://" + forward.addr
	client, err := api.NewClient(config)
	if err != nil {
		forward.Close()
		return nil, nil, fmt.Errorf("create vault client: %w", err)
	}
	client.SetToken(token)

	return client, forward.Close, nil
}

func getLocalVaultToken(ctx context.Context) (string, error) {
	output, err := runKubectl(
		ctx,
		"get", "secret", "vault-unseal-keys",
		"-n", vaultNamespace,
		"-o", `template={{ index .data "vault-root" }}`,
	)
	if err != nil {
		return "", err
	}

	token, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(output)))
	if err != nil {
		return "", fmt.Errorf("decode token: %w", err)
	}
	return string(token), nil
}
