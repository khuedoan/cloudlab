package secrets

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	"golang.org/x/crypto/ssh"
)

const (
	rsaKeyBits       = 4096
	defaultKeyLength = 32
)

type Prompter interface {
	PromptSecret(description string) (string, error)
}

type Generator struct {
	prompter Prompter
}

func NewGenerator(prompter Prompter) *Generator {
	return &Generator{prompter: prompter}
}

func (g *Generator) Generate(e Entry) (map[string]interface{}, error) {
	switch e.Settings.Type {
	case "random":
		length := e.Settings.Length
		if length == 0 {
			length = defaultKeyLength
		}
		value, err := generateRandomString(length)
		if err != nil {
			return nil, fmt.Errorf("generate random string: %w", err)
		}
		log.Info("generated random secret", "path", e.Path, "key", e.DataKey)
		return map[string]interface{}{e.DataKey: value}, nil

	case "ssh":
		algorithm := e.Settings.Algorithm
		if algorithm == "" {
			algorithm = "ed25519"
		}

		privateKey, publicKey, err := generateSSHKeypair(algorithm)
		if err != nil {
			return nil, fmt.Errorf("generate SSH keypair: %w", err)
		}

		pubKeyName := e.Settings.PublicKey
		if pubKeyName == "" {
			pubKeyName = e.DataKey + ".pub"
		}

		log.Info("generated SSH keypair", "algorithm", algorithm, "path", e.Path, "key", e.DataKey)
		return map[string]interface{}{
			e.DataKey:  privateKey,
			pubKeyName: publicKey,
		}, nil

	case "manual":
		if g.prompter == nil {
			return nil, fmt.Errorf("manual secret type requires a prompter")
		}

		description := e.Settings.Description
		if description == "" {
			description = fmt.Sprintf("Enter value for %s#%s", e.Path, e.DataKey)
		}
		value, err := g.prompter.PromptSecret(description)
		if err != nil {
			return nil, fmt.Errorf("prompt for secret: %w", err)
		}
		log.Info("stored manual secret", "path", e.Path, "key", e.DataKey)
		return map[string]interface{}{e.DataKey: value}, nil

	default:
		return nil, fmt.Errorf("unknown secret type: %s", e.Settings.Type)
	}
}

func generateRandomString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	if length <= 0 {
		return "", fmt.Errorf("length must be positive")
	}
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
