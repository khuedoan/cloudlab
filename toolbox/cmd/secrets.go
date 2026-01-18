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
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
	"github.com/hashicorp/vault/api"
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

var settingsFile string

func init() {
	secretsCmd.Flags().StringVar(&settingsFile, "settings", "", "Path to settings YAML file")
	secretsCmd.MarkFlagRequired("settings")
}

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage secrets in Vault",
	RunE:  runSecrets,
}

// SecretsConfig is the root configuration structure
// Format:
//
//	secrets:
//	  secret/path:
//	    KEY_NAME:
//	      type: random|ssh|manual
//	      ...
type SecretsConfig struct {
	Secrets map[string]map[string]SecretSettings `yaml:"secrets"`
}

type SecretSettings struct {
	Type        string `yaml:"type"`
	Length      int    `yaml:"length,omitempty"`
	Algorithm   string `yaml:"algorithm,omitempty"`
	PublicKey   string `yaml:"public_key,omitempty"`
	Description string `yaml:"description,omitempty"`
}

type secretEntry struct {
	path     string
	dataKey  string
	settings SecretSettings
}

func runSecrets(cmd *cobra.Command, args []string) error {
	config, err := loadSecretsConfig(settingsFile)
	if err != nil {
		return fmt.Errorf("load settings file: %w", err)
	}

	entries, err := parseAndValidateConfig(config)
	if err != nil {
		return fmt.Errorf("validate config: %w", err)
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

	var autoEntries, manualEntries []secretEntry
	for _, e := range entries {
		if e.settings.Type == "manual" {
			manualEntries = append(manualEntries, e)
		} else {
			autoEntries = append(autoEntries, e)
		}
	}

	for _, e := range autoEntries {
		if err := processSecret(ctx, client.Vault(), e); err != nil {
			return fmt.Errorf("process secret %s#%s: %w", e.path, e.dataKey, err)
		}
	}

	for _, e := range manualEntries {
		if err := processSecret(ctx, client.Vault(), e); err != nil {
			return fmt.Errorf("process secret %s#%s: %w", e.path, e.dataKey, err)
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

func parseAndValidateConfig(config *SecretsConfig) ([]secretEntry, error) {
	var entries []secretEntry

	paths := make([]string, 0, len(config.Secrets))
	for path := range config.Secrets {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		keys := config.Secrets[path]

		dataKeys := make([]string, 0, len(keys))
		for k := range keys {
			dataKeys = append(dataKeys, k)
		}
		sort.Strings(dataKeys)

		for _, dataKey := range dataKeys {
			settings := keys[dataKey]

			if err := validateSettings(path, dataKey, settings); err != nil {
				return nil, err
			}

			entries = append(entries, secretEntry{
				path:     path,
				dataKey:  dataKey,
				settings: settings,
			})
		}
	}

	return entries, nil
}

func validateSettings(path, dataKey string, settings SecretSettings) error {
	switch settings.Type {
	case "random":
		// valid
	case "ssh":
		// valid
	case "manual":
		// valid
	case "":
		return fmt.Errorf("%s#%s: type is required", path, dataKey)
	default:
		return fmt.Errorf("%s#%s: unknown type %q", path, dataKey, settings.Type)
	}
	return nil
}

func parsePath(fullPath string) (mount, path string, err error) {
	parts := strings.SplitN(fullPath, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid path %q: expected format mount/path", fullPath)
	}
	return parts[0], parts[1], nil
}

func processSecret(ctx context.Context, vault *api.Client, e secretEntry) error {
	mount, path, err := parsePath(e.path)
	if err != nil {
		return err
	}

	existing, _ := vault.KVv2(mount).Get(ctx, path)
	data := make(map[string]interface{})
	if existing != nil && existing.Data != nil {
		for k, v := range existing.Data {
			data[k] = v
		}
	}

	keysToCheck := []string{e.dataKey}
	if e.settings.Type == "ssh" {
		pubKey := e.settings.PublicKey
		if pubKey == "" {
			pubKey = e.dataKey + ".pub"
		}
		keysToCheck = append(keysToCheck, pubKey)
	}

	allExist := true
	for _, k := range keysToCheck {
		if _, exists := data[k]; !exists {
			allExist = false
			break
		}
	}
	if allExist {
		log.Info("secret already exists, skipping", "path", e.path, "key", e.dataKey)
		return nil
	}

	newData, err := generateSecretData(e)
	if err != nil {
		return err
	}

	for k, v := range newData {
		data[k] = v
	}

	if _, err := vault.KVv2(mount).Put(ctx, path, data); err != nil {
		return fmt.Errorf("write to Vault: %w", err)
	}

	return nil
}

func generateSecretData(e secretEntry) (map[string]interface{}, error) {
	switch e.settings.Type {
	case "random":
		length := e.settings.Length
		if length == 0 {
			length = defaultKeyLength
		}
		value, err := generateRandomString(length)
		if err != nil {
			return nil, fmt.Errorf("generate random string: %w", err)
		}
		log.Info("generated random secret", "path", e.path, "key", e.dataKey)
		return map[string]interface{}{e.dataKey: value}, nil

	case "ssh":
		algorithm := e.settings.Algorithm
		if algorithm == "" {
			algorithm = "ed25519"
		}
		privateKey, publicKey, err := generateSSHKeypair(algorithm)
		if err != nil {
			return nil, fmt.Errorf("generate SSH keypair: %w", err)
		}

		pubKeyName := e.settings.PublicKey
		if pubKeyName == "" {
			pubKeyName = e.dataKey + ".pub"
		}

		log.Info("generated SSH keypair", "algorithm", algorithm, "path", e.path, "key", e.dataKey)
		return map[string]interface{}{
			e.dataKey:  privateKey,
			pubKeyName: publicKey,
		}, nil

	case "manual":
		description := e.settings.Description
		if description == "" {
			description = fmt.Sprintf("Enter value for %s#%s", e.path, e.dataKey)
		}
		value, err := promptForSecret(description)
		if err != nil {
			return nil, fmt.Errorf("prompt for secret: %w", err)
		}
		log.Info("stored manual secret", "path", e.path, "key", e.dataKey)
		return map[string]interface{}{e.dataKey: value}, nil

	default:
		return nil, fmt.Errorf("unknown secret type: %s", e.settings.Type)
	}
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
