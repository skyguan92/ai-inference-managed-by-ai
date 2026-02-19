package docker

import (
	"context"
	"fmt"
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

	// Add restart policy
	args = append(args, "--restart", "unless-stopped")

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

	// Remove the container
	rmCmd := exec.CommandContext(ctx, "docker", "rm", containerID)
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
	args := []string{"ps", "-q", "--filter", "label=aima.managed=true"}

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

// CheckDocker checks if Docker is available
func CheckDocker() error {
	cmd := exec.Command("docker", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker is not available: %w", err)
	}
	return nil
}
