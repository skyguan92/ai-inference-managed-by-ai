package docker

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// SimpleClient is a lightweight Docker client using CLI commands
type SimpleClient struct{}

// NewSimpleClient creates a new simple Docker client
func NewSimpleClient() *SimpleClient {
	return &SimpleClient{}
}

// PullImage pulls a Docker image
func (c *SimpleClient) PullImage(ctx context.Context, image string) error {
	cmd := exec.CommandContext(ctx, "docker", "pull", image)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker pull %s failed: %w\nOutput: %s", image, err, string(output))
	}
	return nil
}

// CreateAndStartContainer creates and starts a container
func (c *SimpleClient) CreateAndStartContainer(ctx context.Context, name, image string, opts ContainerOptions) (string, error) {
	args := []string{"run", "-d", "--name", name}

	// Add port mappings
	for hostPort, containerPort := range opts.Ports {
		args = append(args, "-p", fmt.Sprintf("%s:%s", hostPort, containerPort))
	}

	// Add volumes
	for hostPath, containerPath := range opts.Volumes {
		args = append(args, "-v", fmt.Sprintf("%s:%s", hostPath, containerPath))
	}

	// Add environment variables
	for _, env := range opts.Env {
		args = append(args, "-e", env)
	}

	// Add labels
	for k, v := range opts.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", k, v))
	}

	// Add GPU support if requested
	if opts.GPU {
		args = append(args, "--gpus", "all")
	}

	// Add working directory
	if opts.WorkingDir != "" {
		args = append(args, "-w", opts.WorkingDir)
	}

	// Add resource limits
	if opts.Memory != "" {
		args = append(args, "--memory", opts.Memory)
	}
	if opts.CPU != "" {
		args = append(args, "--cpus", opts.CPU)
	}

	// Add image and command
	args = append(args, image)
	if len(opts.Cmd) > 0 {
		args = append(args, opts.Cmd...)
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker run failed: %w\nOutput: %s", err, string(output))
	}

	// Container ID is the output
	containerID := strings.TrimSpace(string(output))
	return containerID, nil
}

// StopContainer stops and removes a container
func (c *SimpleClient) StopContainer(ctx context.Context, containerID string, timeout int) error {
	// Stop the container
	stopCmd := exec.CommandContext(ctx, "docker", "stop", "-t", fmt.Sprintf("%d", timeout), containerID)
	if output, err := stopCmd.CombinedOutput(); err != nil {
		// Ignore errors, container might already be stopped
		_ = output
	}

	// Remove the container using a fresh context — the request context may have
	// expired while docker stop was running (especially with long timeouts).
	rmCmd := exec.CommandContext(context.Background(), "docker", "rm", containerID)
	if output, err := rmCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker rm failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// GetContainerStatus gets container status
func (c *SimpleClient) GetContainerStatus(ctx context.Context, containerID string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Status}}", containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker inspect failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetContainerLogs gets container logs
func (c *SimpleClient) GetContainerLogs(ctx context.Context, containerID string, tail int) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "logs", "--tail", fmt.Sprintf("%d", tail), containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker logs failed: %w", err)
	}
	return string(output), nil
}

// ListContainers lists containers with given labels
func (c *SimpleClient) ListContainers(ctx context.Context, labels map[string]string) ([]string, error) {
	args := []string{"ps", "-a", "-q", "--filter", "label=aima.managed=true"}

	for k, v := range labels {
		args = append(args, "--filter", fmt.Sprintf("label=%s=%s", k, v))
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker ps failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var containers []string
	for _, line := range lines {
		if line != "" {
			containers = append(containers, line)
		}
	}
	return containers, nil
}

// FindContainersByPort returns all containers publishing the given host port.
// Uses `docker ps -a --filter publish=PORT` which finds ANY container on that
// port regardless of labels, enabling orphan detection.
func (c *SimpleClient) FindContainersByPort(ctx context.Context, port int) ([]PortConflict, error) {
	args := []string{
		"ps", "-a",
		"--filter", fmt.Sprintf("publish=%d", port),
		"--format", "{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Labels}}",
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker ps (port filter): %w\nOutput: %s", err, string(output))
	}

	var conflicts []PortConflict
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 3 {
			continue
		}
		labels := ""
		if len(parts) == 4 {
			labels = parts[3]
		}
		// docker ps --format {{.Labels}} uses "key=value,key=value" (Docker CLI >=20)
		// but may fall back to Go's map format "map[key:value key:value]" on older
		// versions. Check both separator styles to be version-resilient.
		isAIMA := strings.Contains(labels, "aima.managed=true") ||
			strings.Contains(labels, "aima.managed:true")
		conflicts = append(conflicts, PortConflict{
			ContainerID: parts[0],
			Name:        parts[1],
			Image:       parts[2],
			IsAIMA:      isAIMA,
		})
	}
	return conflicts, nil
}

// CheckDocker checks if Docker is available
func CheckDocker() error {
	cmd := exec.Command("docker", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker is not available: %w", err)
	}
	return nil
}

// StreamLogs streams container log lines to out, starting from since (RFC3339 or relative like "1h").
// Blocks until ctx is cancelled or the container exits.
func (c *SimpleClient) StreamLogs(ctx context.Context, containerID string, since string, out chan<- string) error {
	args := []string{"logs", "-f", "--tail", "0"}
	if since != "" {
		args = append(args, "--since", since)
	}
	args = append(args, containerID)

	cmd := exec.CommandContext(ctx, "docker", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("docker logs pipe failed: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("docker logs stderr pipe failed: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("docker logs start failed: %w", err)
	}

	forward := func(r io.Reader) {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			select {
			case out <- scanner.Text():
			case <-ctx.Done():
				return
			}
		}
	}

	go forward(stdout)
	go forward(stderr)

	if err := cmd.Wait(); err != nil {
		// Context cancellation is expected; treat it as a clean exit.
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("docker logs exited: %w", err)
	}
	return nil
}

// ContainerEvents returns a channel of Docker container events matching filters.
// The channel is closed when ctx is cancelled.
func (c *SimpleClient) ContainerEvents(ctx context.Context, filters map[string]string) (<-chan ContainerEvent, error) {
	args := []string{"events", "--format", "{{.ID}} {{.Action}} {{.Status}}", "--filter", "type=container"}
	for k, v := range filters {
		args = append(args, "--filter", fmt.Sprintf("%s=%s", k, v))
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("docker events pipe failed: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("docker events start failed: %w", err)
	}

	ch := make(chan ContainerEvent, 16)
	go func() {
		defer close(ch)
		defer cmd.Wait() //nolint:errcheck
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			parts := strings.SplitN(scanner.Text(), " ", 3)
			if len(parts) < 2 {
				continue
			}
			ev := ContainerEvent{
				ContainerID: parts[0],
				Action:      parts[1],
			}
			if len(parts) == 3 {
				ev.Status = parts[2]
			} else {
				ev.Status = parts[1]
			}
			select {
			case ch <- ev:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}
