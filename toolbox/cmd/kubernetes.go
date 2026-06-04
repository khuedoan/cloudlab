package cmd

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

type kubectlPortForward struct {
	addr   string
	cancel context.CancelFunc
	done   chan error
	output *bytes.Buffer
}

func runKubectl(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	return cmd.CombinedOutput()
}

func runKubectlInput(ctx context.Context, input []byte, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Stdin = bytes.NewReader(input)
	return cmd.CombinedOutput()
}

func startKubectlPortForward(ctx context.Context, namespace, resource string, remotePort int) (*kubectlPortForward, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("reserve local port: %w", err)
	}
	localPort := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		return nil, fmt.Errorf("release local port: %w", err)
	}

	portForwardCtx, cancel := context.WithCancel(ctx)
	output := &bytes.Buffer{}
	cmd := exec.CommandContext(
		portForwardCtx,
		"kubectl",
		"-n", namespace,
		"port-forward",
		resource,
		fmt.Sprintf("%d:%d", localPort, remotePort),
	)
	cmd.Stdout = output
	cmd.Stderr = output

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start kubectl port-forward: %w", err)
	}

	forward := &kubectlPortForward{
		addr:   fmt.Sprintf("127.0.0.1:%d", localPort),
		cancel: cancel,
		done:   make(chan error, 1),
		output: output,
	}
	go func() {
		forward.done <- cmd.Wait()
	}()

	if err := forward.waitReady(ctx); err != nil {
		forward.Close()
		return nil, err
	}

	return forward, nil
}

func (f *kubectlPortForward) waitReady(ctx context.Context) error {
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		conn, err := net.DialTimeout("tcp", f.addr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}

		select {
		case err := <-f.done:
			f.done <- err
			return fmt.Errorf("kubectl port-forward exited early: %w (output: %s)", err, strings.TrimSpace(f.output.String()))
		case <-ctx.Done():
			return fmt.Errorf("wait for kubectl port-forward: %w", ctx.Err())
		case <-deadline.C:
			return fmt.Errorf("wait for kubectl port-forward to %s timed out (output: %s)", f.addr, strings.TrimSpace(f.output.String()))
		case <-ticker.C:
		}
	}
}

func (f *kubectlPortForward) Close() {
	f.cancel()
	<-f.done
}
