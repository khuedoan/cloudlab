package secrets

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/vault/api"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

const (
	localPort      = 18200
	remotePort     = 8200
	vaultNamespace = "vault"
	vaultService   = "svc/vault"
	connectTimeout = 30 * time.Second
	retryInterval  = 1 * time.Second
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

// HostInfo represents a host entry in hosts.json
type HostInfo struct {
	IPv6Address string `json:"ipv6_address"`
}

// Run executes the secrets subcommand
func Run() {
	var hostsFile, host, sshUser, sshKey, specFile string
	flag.StringVar(&hostsFile, "hosts-file", "", "Path to hosts.json file")
	flag.StringVar(&host, "host", "", "Host name to connect to (e.g., kube-1)")
	flag.StringVar(&sshUser, "ssh-user", "root", "SSH user")
	flag.StringVar(&sshKey, "ssh-key", "", "Path to SSH private key (default: ~/.ssh/id_ed25519)")
	flag.StringVar(&specFile, "spec", "", "Path to secrets specification YAML file")
	flag.Parse()

	if hostsFile == "" {
		log.Fatal("--hosts-file must be provided")
	}
	if host == "" {
		log.Fatal("--host must be provided")
	}
	if specFile == "" {
		log.Fatal("--spec must be provided")
	}

	// Default SSH key path
	if sshKey == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("failed to get home directory: %v", err)
		}
		sshKey = filepath.Join(home, ".ssh", "id_ed25519")
	}

	// Load spec file
	config, err := loadConfig(specFile)
	if err != nil {
		log.Fatalf("failed to load spec file: %v", err)
	}

	// Load hosts file and get IPv6 address
	ipv6Addr, err := loadHostsFile(hostsFile, host)
	if err != nil {
		log.Fatalf("failed to load hosts file: %v", err)
	}
	log.Printf("resolved host %q to %s", host, ipv6Addr)

	// Connect via SSH
	sshClient, err := connectSSH(ipv6Addr, sshUser, sshKey)
	if err != nil {
		log.Fatalf("failed to connect via SSH: %v", err)
	}
	defer sshClient.Close()
	log.Printf("connected to %s@%s", sshUser, ipv6Addr)

	// Get Vault root token
	vaultToken, err := getVaultToken(sshClient)
	if err != nil {
		log.Fatalf("failed to get Vault token: %v", err)
	}
	log.Println("retrieved Vault root token")

	// Start kubectl port-forward in background
	portForwardDone := make(chan error, 1)
	go func() {
		portForwardDone <- runKubectlPortForward(sshClient)
	}()

	// Give kubectl port-forward a moment to start
	time.Sleep(500 * time.Millisecond)

	// Check if port-forward failed immediately
	select {
	case err := <-portForwardDone:
		log.Fatalf("kubectl port-forward failed: %v", err)
	default:
		// Still running, continue
	}

	// Set up SSH tunnel: local:18200 -> remote:8200
	tunnelCtx, tunnelCancel := context.WithCancel(context.Background())
	defer tunnelCancel()

	listener, err := setupLocalListener()
	if err != nil {
		log.Fatalf("failed to set up local listener: %v", err)
	}
	defer listener.Close()

	go runSSHTunnel(tunnelCtx, sshClient, listener)
	log.Printf("SSH tunnel established: localhost:%d -> remote:%d", localPort, remotePort)

	// Wait for Vault to be reachable
	vaultAddr := fmt.Sprintf("http://127.0.0.1:%d", localPort)
	vaultClient, err := waitForVault(vaultAddr, vaultToken, connectTimeout)
	if err != nil {
		log.Fatalf("failed to connect to Vault: %v", err)
	}
	log.Println("connected to Vault")

	ctx := context.Background()

	// Process each secret
	for _, spec := range config.Secrets {
		if err := processSecret(ctx, vaultClient, spec); err != nil {
			log.Fatalf("failed to process secret %q: %v", spec.Path, err)
		}
	}

	log.Println("all secrets processed successfully")
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	return &config, nil
}

func loadHostsFile(path, hostName string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}

	var hosts map[string]HostInfo
	if err := json.Unmarshal(data, &hosts); err != nil {
		return "", fmt.Errorf("parsing JSON: %w", err)
	}

	hostInfo, ok := hosts[hostName]
	if !ok {
		available := make([]string, 0, len(hosts))
		for name := range hosts {
			available = append(available, name)
		}
		return "", fmt.Errorf("host %q not found in hosts file (available: %s)", hostName, strings.Join(available, ", "))
	}

	if hostInfo.IPv6Address == "" {
		return "", fmt.Errorf("host %q has no ipv6_address", hostName)
	}

	return hostInfo.IPv6Address, nil
}

func connectSSH(ipv6Addr, user, keyPath string) (*ssh.Client, error) {
	// Read SSH private key
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("reading SSH key %s: %w", keyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("parsing SSH key: %w", err)
	}

	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: consider proper host key verification
		Timeout:         10 * time.Second,
	}

	// IPv6 addresses need brackets
	addr := fmt.Sprintf("[%s]:22", ipv6Addr)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("dialing %s: %w", addr, err)
	}

	return client, nil
}

func getVaultToken(sshClient *ssh.Client) (string, error) {
	session, err := sshClient.NewSession()
	if err != nil {
		return "", fmt.Errorf("creating session: %w", err)
	}
	defer session.Close()

	cmd := fmt.Sprintf(
		`kubectl get secret vault-unseal-keys -n %s -o template='{{ index .data "vault-root" }}'`,
		vaultNamespace,
	)

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return "", fmt.Errorf("running kubectl: %w\noutput: %s", err, string(output))
	}

	// Decode base64
	token, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(output)))
	if err != nil {
		return "", fmt.Errorf("decoding token: %w", err)
	}

	return string(token), nil
}

func runKubectlPortForward(sshClient *ssh.Client) error {
	session, err := sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("creating session: %w", err)
	}
	defer session.Close()

	cmd := fmt.Sprintf(
		"kubectl port-forward %s -n %s %d:%d",
		vaultService, vaultNamespace, remotePort, remotePort,
	)

	// This will block until the port-forward exits
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("kubectl port-forward failed: %w\noutput: %s", err, string(output))
	}

	return nil
}

func setupLocalListener() (net.Listener, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", localPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listening on %s: %w (port may be in use)", addr, err)
	}
	return listener, nil
}

func runSSHTunnel(ctx context.Context, sshClient *ssh.Client, listener net.Listener) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		localConn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("tunnel: failed to accept connection: %v", err)
			continue
		}

		go handleTunnelConnection(ctx, sshClient, localConn)
	}
}

func handleTunnelConnection(ctx context.Context, sshClient *ssh.Client, localConn net.Conn) {
	defer localConn.Close()

	// Connect to remote port through SSH
	remoteAddr := fmt.Sprintf("127.0.0.1:%d", remotePort)
	remoteConn, err := sshClient.Dial("tcp", remoteAddr)
	if err != nil {
		log.Printf("tunnel: failed to connect to remote %s: %v", remoteAddr, err)
		return
	}
	defer remoteConn.Close()

	// Bidirectional copy
	done := make(chan struct{}, 2)

	go func() {
		io.Copy(remoteConn, localConn)
		done <- struct{}{}
	}()

	go func() {
		io.Copy(localConn, remoteConn)
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
	case <-done:
	}
}

func waitForVault(addr, token string, timeout time.Duration) (*api.Client, error) {
	config := api.DefaultConfig()
	config.Address = addr

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}
	client.SetToken(token)

	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		// Try to check Vault health
		_, err := client.Sys().Health()
		if err == nil {
			return client, nil
		}
		lastErr = err
		time.Sleep(retryInterval)
	}

	return nil, fmt.Errorf("timeout after %v waiting for Vault: %w", timeout, lastErr)
}

func processSecret(ctx context.Context, client *api.Client, spec SecretSpec) error {
	// Parse the path to extract mount and secret path
	// Expected format: secret/path/to/secret (where "secret" is the mount)
	mount, secretPath := parsePath(spec.Path)

	// Check if secret exists (KV v2)
	existing, err := client.KVv2(mount).Get(ctx, secretPath)
	if err == nil && existing != nil {
		log.Printf("secret %q already exists, skipping", spec.Path)
		return nil
	}

	// Generate or prompt for secret value
	var secretData map[string]interface{}

	switch spec.Type {
	case "random":
		length := spec.Length
		if length == 0 {
			length = 32
		}
		value, err := generateRandomString(length)
		if err != nil {
			return fmt.Errorf("generating random string: %w", err)
		}
		secretData = map[string]interface{}{
			"value": value,
		}
		log.Printf("generated random secret for %q", spec.Path)

	case "ssh":
		algorithm := spec.Algorithm
		if algorithm == "" {
			algorithm = "ed25519"
		}
		privateKey, publicKey, err := generateSSHKeypair(algorithm)
		if err != nil {
			return fmt.Errorf("generating SSH keypair: %w", err)
		}
		secretData = map[string]interface{}{
			"private_key": privateKey,
			"public_key":  publicKey,
		}
		log.Printf("generated SSH keypair (%s) for %q", algorithm, spec.Path)

	case "manual":
		description := spec.Description
		if description == "" {
			description = fmt.Sprintf("Enter value for %s", spec.Path)
		}
		value, err := promptForSecret(description)
		if err != nil {
			return fmt.Errorf("prompting for secret: %w", err)
		}
		secretData = map[string]interface{}{
			"value": value,
		}
		log.Printf("stored manual secret for %q", spec.Path)

	default:
		return fmt.Errorf("unknown secret type: %s", spec.Type)
	}

	// Write secret to Vault (KV v2)
	_, err = client.KVv2(mount).Put(ctx, secretPath, secretData)
	if err != nil {
		return fmt.Errorf("writing to Vault: %w", err)
	}

	return nil
}

// parsePath extracts the mount and secret path from a full path
// e.g., "secret/myapp/db-password" -> mount="secret", path="myapp/db-password"
func parsePath(fullPath string) (mount, path string) {
	parts := strings.SplitN(fullPath, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "secret", fullPath
}

func generateRandomString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)

	for i := range result {
		b := make([]byte, 1)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		result[i] = charset[int(b[0])%len(charset)]
	}

	return string(result), nil
}

func generateSSHKeypair(algorithm string) (privateKeyPEM, publicKeyOpenSSH string, err error) {
	switch algorithm {
	case "ed25519":
		return generateED25519Keypair()
	case "rsa":
		return generateRSAKeypair(4096)
	default:
		return "", "", fmt.Errorf("unsupported algorithm: %s", algorithm)
	}
}

func generateED25519Keypair() (privateKeyPEM, publicKeyOpenSSH string, err error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generating key: %w", err)
	}

	// Marshal private key to PEM
	privKeyBytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return "", "", fmt.Errorf("marshaling private key: %w", err)
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	// Convert to SSH public key format
	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return "", "", fmt.Errorf("converting to SSH public key: %w", err)
	}

	pubKeyStr := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPubKey)))

	return string(privPEM), pubKeyStr, nil
}

func generateRSAKeypair(bits int) (privateKeyPEM, publicKeyOpenSSH string, err error) {
	privKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return "", "", fmt.Errorf("generating key: %w", err)
	}

	// Marshal private key to PEM
	privKeyBytes := x509.MarshalPKCS1PrivateKey(privKey)
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	// Convert to SSH public key format
	sshPubKey, err := ssh.NewPublicKey(&privKey.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("converting to SSH public key: %w", err)
	}

	pubKeyStr := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPubKey)))

	return string(privPEM), pubKeyStr, nil
}

func promptForSecret(description string) (string, error) {
	fmt.Printf("%s: ", description)

	// Check if stdin is a terminal
	if term.IsTerminal(int(os.Stdin.Fd())) {
		// Read password without echo
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println() // Print newline after hidden input
		if err != nil {
			return "", fmt.Errorf("reading password: %w", err)
		}
		return string(password), nil
	}

	// Non-terminal input (piped)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}

	return "", fmt.Errorf("no input provided")
}
