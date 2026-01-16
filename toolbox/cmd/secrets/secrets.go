package secrets

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
	"github.com/hashicorp/vault/api"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"

	"toolbox/internal/cluster"
)

const (
	vaultNamespace   = "vault"
	vaultService     = "svc/vault"
	vaultPort        = 8200
	connectTimeout   = 30 * time.Second
	retryInterval    = 1 * time.Second
	rsaKeyBits       = 4096
	defaultKeyLength = 32
)

// SecretSpec defines a single secret specification
type SecretSpec struct {
	Path        string `yaml:"path"`
	Type        string `yaml:"type"` // random, ssh, manual
	Length      int    `yaml:"length,omitempty"`
	Algorithm   string `yaml:"algorithm,omitempty"` // ed25519, rsa (for ssh type)
	Description string `yaml:"description,omitempty"`
}

// Config is the root configuration structure
type Config struct {
	Secrets []SecretSpec `yaml:"secrets"`
}

func init() {
	log.SetReportTimestamp(false)
}

// Run executes the secrets subcommand
func Run() error {
	var hostsFile, host, sshUser, sshKey, specFile string
	flag.StringVar(&hostsFile, "hosts-file", "", "Path to hosts.json file")
	flag.StringVar(&host, "host", "", "Host name to connect to (e.g., kube-1)")
	flag.StringVar(&sshUser, "ssh-user", "root", "SSH user")
	flag.StringVar(&sshKey, "ssh-key", "", "Path to SSH private key (default: ~/.ssh/id_ed25519)")
	flag.StringVar(&specFile, "spec", "", "Path to secrets specification YAML file")
	flag.Parse()

	// Validate required flags
	required := map[string]string{
		"hosts-file": hostsFile,
		"host":       host,
		"spec":       specFile,
	}
	for name, val := range required {
		if val == "" {
			return fmt.Errorf("--%s must be provided", name)
		}
	}

	// Default SSH key path
	if sshKey == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("get home directory: %w", err)
		}
		sshKey = filepath.Join(home, ".ssh", "id_ed25519")
	}

	config, err := loadConfig(specFile)
	if err != nil {
		return fmt.Errorf("load spec file: %w", err)
	}

	hostAddr, err := cluster.LoadHost(hostsFile, host)
	if err != nil {
		return fmt.Errorf("load hosts file: %w", err)
	}
	log.Debug("resolved host", "host", host, "addr", hostAddr)

	conn, err := cluster.Connect(
		cluster.SSHConfig{Host: hostAddr, User: sshUser, KeyPath: sshKey},
		cluster.ServiceConfig{Namespace: vaultNamespace, Name: vaultService, Port: vaultPort},
	)
	if err != nil {
		return fmt.Errorf("connect to cluster: %w", err)
	}
	defer conn.Close()
	log.Debug("connected to cluster", "addr", conn.LocalAddr)

	vaultToken, err := getVaultToken(conn)
	if err != nil {
		return fmt.Errorf("get Vault token: %w", err)
	}
	log.Debug("retrieved Vault root token")

	vaultAddr := "http://" + conn.LocalAddr
	vaultClient, err := waitForVault(vaultAddr, vaultToken, connectTimeout)
	if err != nil {
		return fmt.Errorf("connect to Vault: %w", err)
	}
	log.Debug("connected to Vault")

	// Process secrets
	ctx := context.Background()
	for _, spec := range config.Secrets {
		if err := processSecret(ctx, vaultClient, spec); err != nil {
			return fmt.Errorf("process secret %q: %w", spec.Path, err)
		}
	}

	log.Info("all secrets processed successfully")
	return nil
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	return &config, nil
}

func getVaultToken(conn *cluster.Connector) (string, error) {
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

func waitForVault(addr, token string, timeout time.Duration) (*api.Client, error) {
	config := api.DefaultConfig()
	config.Address = addr

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}
	client.SetToken(token)

	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		if _, err := client.Sys().Health(); err == nil {
			return client, nil
		} else {
			lastErr = err
		}
		time.Sleep(retryInterval)
	}

	return nil, fmt.Errorf("timeout after %v: %w", timeout, lastErr)
}

func processSecret(ctx context.Context, client *api.Client, spec SecretSpec) error {
	mount, secretPath, err := parsePath(spec.Path)
	if err != nil {
		return err
	}

	existing, err := client.KVv2(mount).Get(ctx, secretPath)
	if err == nil && existing != nil {
		log.Info("secret already exists, skipping", "path", spec.Path)
		return nil
	}

	secretData, err := generateSecretData(spec)
	if err != nil {
		return err
	}

	if _, err := client.KVv2(mount).Put(ctx, secretPath, secretData); err != nil {
		return fmt.Errorf("write to Vault: %w", err)
	}

	return nil
}

func generateSecretData(spec SecretSpec) (map[string]interface{}, error) {
	switch spec.Type {
	case "random":
		length := spec.Length
		if length == 0 {
			length = defaultKeyLength
		}
		value, err := generateRandomString(length)
		if err != nil {
			return nil, fmt.Errorf("generate random string: %w", err)
		}
		log.Info("generated random secret", "path", spec.Path)
		return map[string]interface{}{"value": value}, nil

	case "ssh":
		algorithm := spec.Algorithm
		if algorithm == "" {
			algorithm = "ed25519"
		}
		privateKey, publicKey, err := generateSSHKeypair(algorithm)
		if err != nil {
			return nil, fmt.Errorf("generate SSH keypair: %w", err)
		}
		log.Info("generated SSH keypair", "algorithm", algorithm, "path", spec.Path)
		return map[string]interface{}{
			"private_key": privateKey,
			"public_key":  publicKey,
		}, nil

	case "manual":
		description := spec.Description
		if description == "" {
			description = fmt.Sprintf("Enter value for %s", spec.Path)
		}
		value, err := promptForSecret(description)
		if err != nil {
			return nil, fmt.Errorf("prompt for secret: %w", err)
		}
		log.Info("stored manual secret", "path", spec.Path)
		return map[string]interface{}{"value": value}, nil

	default:
		return nil, fmt.Errorf("unknown secret type: %s", spec.Type)
	}
}

// parsePath extracts the mount and secret path from a full path.
// Expected format: mount/path/to/secret (e.g., "secret/myapp/db-password")
func parsePath(fullPath string) (mount, path string, err error) {
	parts := strings.SplitN(fullPath, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid path %q: expected format mount/path", fullPath)
	}
	return parts[0], parts[1], nil
}

func generateRandomString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	randomBytes := make([]byte, length)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	for i, b := range randomBytes {
		result[i] = charset[int(b)%len(charset)]
	}

	return string(result), nil
}

func generateSSHKeypair(algorithm string) (privateKeyPEM, publicKeyOpenSSH string, err error) {
	switch algorithm {
	case "ed25519":
		pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return "", "", fmt.Errorf("generate key: %w", err)
		}
		privKeyBytes, err := x509.MarshalPKCS8PrivateKey(privKey)
		if err != nil {
			return "", "", fmt.Errorf("marshal private key: %w", err)
		}
		privPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: privKeyBytes,
		})
		pubKeyStr, err := marshalSSHPublicKey(pubKey)
		if err != nil {
			return "", "", err
		}
		return string(privPEM), pubKeyStr, nil

	case "rsa":
		privKey, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
		if err != nil {
			return "", "", fmt.Errorf("generate key: %w", err)
		}
		privPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privKey),
		})
		pubKeyStr, err := marshalSSHPublicKey(&privKey.PublicKey)
		if err != nil {
			return "", "", err
		}
		return string(privPEM), pubKeyStr, nil

	default:
		return "", "", fmt.Errorf("unsupported algorithm: %s", algorithm)
	}
}

func marshalSSHPublicKey(key interface{}) (string, error) {
	sshPubKey, err := ssh.NewPublicKey(key)
	if err != nil {
		return "", fmt.Errorf("convert to SSH public key: %w", err)
	}
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPubKey))), nil
}

func promptForSecret(description string) (string, error) {
	var value string

	err := huh.NewInput().
		Title(description).
		EchoMode(huh.EchoModePassword).
		Value(&value).
		Run()

	if err != nil {
		return "", fmt.Errorf("prompt: %w", err)
	}

	if value == "" {
		return "", fmt.Errorf("no input provided")
	}

	return value, nil
}
