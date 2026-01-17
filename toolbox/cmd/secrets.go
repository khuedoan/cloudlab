package cmd

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"

	"toolbox/internal/cluster"
)

const (
	rsaKeyBits       = 4096
	defaultKeyLength = 32
	connectTimeout   = 30 * time.Second
)

var specFile string

func init() {
	secretsCmd.Flags().StringVar(&specFile, "spec", "", "Path to secrets specification YAML file")
	secretsCmd.MarkFlagRequired("spec")
}

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage secrets in Vault",
	RunE:  runSecrets,
}

type SecretSpec struct {
	Path        string `yaml:"path"`
	Type        string `yaml:"type"`
	Length      int    `yaml:"length,omitempty"`
	Algorithm   string `yaml:"algorithm,omitempty"`
	Description string `yaml:"description,omitempty"`
}

type SecretsConfig struct {
	Secrets []SecretSpec `yaml:"secrets"`
}

func runSecrets(cmd *cobra.Command, args []string) error {
	config, err := loadSecretsConfig(specFile)
	if err != nil {
		return fmt.Errorf("load spec file: %w", err)
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), connectTimeout)
	defer cancel()

	client, err := cluster.NewClient(ctx, cluster.ClientConfig{
		HostsFile: hostsFile,
		Host:      host,
		SSHUser:   sshUser,
		SSHKey:    sshKey,
	})
	if err != nil {
		return fmt.Errorf("connect to cluster: %w", err)
	}
	defer client.Close()
	log.Debug("connected to cluster")

	for _, spec := range config.Secrets {
		if err := processSecret(ctx, client, spec); err != nil {
			return fmt.Errorf("process secret %q: %w", spec.Path, err)
		}
	}

	log.Info("all secrets processed successfully")
	return nil
}

func loadSecretsConfig(path string) (*SecretsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var config SecretsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	return &config, nil
}

func processSecret(ctx context.Context, client *cluster.Client, spec SecretSpec) error {
	mount, secretPath, err := parsePath(spec.Path)
	if err != nil {
		return err
	}

	existing, err := client.Vault().KVv2(mount).Get(ctx, secretPath)
	if err == nil && existing != nil {
		log.Info("secret already exists, skipping", "path", spec.Path)
		return nil
	}

	secretData, err := generateSecretData(spec)
	if err != nil {
		return err
	}

	if _, err := client.Vault().KVv2(mount).Put(ctx, secretPath, secretData); err != nil {
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
