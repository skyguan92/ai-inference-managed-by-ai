package docker

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// SDKClient implements Client using the official Docker Go SDK.
type SDKClient struct {
	cli *dockerclient.Client
}

// NewSDKClient creates an SDKClient configured from environment variables
// (DOCKER_HOST, DOCKER_TLS_VERIFY, DOCKER_CERT_PATH, DOCKER_API_VERSION).
func NewSDKClient() (*SDKClient, error) {
	cli, err := dockerclient.NewClientWithOpts(
		dockerclient.FromEnv,
		dockerclient.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("docker sdk client: %w", err)
	}
	return &SDKClient{cli: cli}, nil
}

// CreateAndStartContainer creates and starts a container, returning its ID.
func (c *SDKClient) CreateAndStartContainer(ctx context.Context, name, image string, opts ContainerOptions) (string, error) {
	// Build port bindings from opts.Ports (hostPort -> containerPort).
	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}
	for hostPort, containerPort := range opts.Ports {
		p := nat.Port(containerPort + "/tcp")
		exposedPorts[p] = struct{}{}
		portBindings[p] = []nat.PortBinding{{HostPort: hostPort}}
	}

	// Build volume bindings from opts.Volumes (hostPath -> containerPath).
	binds := make([]string, 0, len(opts.Volumes))
	for hostPath, containerPath := range opts.Volumes {
		binds = append(binds, hostPath+":"+containerPath)
	}

	// Build label map.
	labels := make(map[string]string, len(opts.Labels))
	for k, v := range opts.Labels {
		labels[k] = v
	}

	// Container config.
	cfg := &container.Config{
		Image:        image,
		Cmd:          opts.Cmd,
		Env:          opts.Env,
		Labels:       labels,
		ExposedPorts: exposedPorts,
		WorkingDir:   opts.WorkingDir,
	}

	// Host config.
	hostCfg := &container.HostConfig{
		Binds:        binds,
		PortBindings: portBindings,
		// No automatic restart — AIMA handles retries in HybridEngineProvider.
		// Using "unless-stopped" caused false "restarting" status during health
		// checks when port binding failed between retries.
	}

	// Memory limit.
	if opts.Memory != "" {
		mem, err := parseMemory(opts.Memory)
		if err == nil {
			hostCfg.Memory = mem
		}
	}

	// CPU limit.
	if opts.CPU != "" {
		if cpus, err := strconv.ParseFloat(opts.CPU, 64); err == nil {
			hostCfg.NanoCPUs = int64(cpus * 1e9)
		}
	}

	// GPU support — equivalent to `--gpus all`.
	if opts.GPU {
		hostCfg.DeviceRequests = []container.DeviceRequest{
			{
				Driver:       "nvidia",
				Count:        -1, // all GPUs
				Capabilities: [][]string{{"gpu"}},
			},
		}
	}

	resp, err := c.cli.ContainerCreate(ctx, cfg, hostCfg, nil, nil, name)
	if err != nil {
		return "", fmt.Errorf("docker ContainerCreate: %w", err)
	}

	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		// Clean up the created container so it doesn't block the port on retry.
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = c.cli.ContainerRemove(cleanupCtx, resp.ID, container.RemoveOptions{Force: true})
		return "", fmt.Errorf("docker ContainerStart: %w", err)
	}

	return resp.ID, nil
}

// StopContainer stops and removes a container. timeout is in seconds.
func (c *SDKClient) StopContainer(ctx context.Context, containerID string, timeout int) error {
	stopOpts := container.StopOptions{Timeout: &timeout}
	if err := c.cli.ContainerStop(ctx, containerID, stopOpts); err != nil {
		// Ignore "not found" or "not running" errors — container may already be gone.
		if !cerrdefs.IsNotFound(err) {
			return fmt.Errorf("docker ContainerStop: %w", err)
		}
	}

	if err := c.cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true}); err != nil {
		if !cerrdefs.IsNotFound(err) {
			return fmt.Errorf("docker ContainerRemove: %w", err)
		}
	}
	return nil
}

// GetContainerStatus returns the container state string (e.g. "running", "exited").
func (c *SDKClient) GetContainerStatus(ctx context.Context, containerID string) (string, error) {
	info, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("docker ContainerInspect: %w", err)
	}
	return info.State.Status, nil
}

// GetContainerLogs returns the last tail lines of container logs (stdout+stderr combined).
func (c *SDKClient) GetContainerLogs(ctx context.Context, containerID string, tail int) (string, error) {
	logOpts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       strconv.Itoa(tail),
	}
	rc, err := c.cli.ContainerLogs(ctx, containerID, logOpts)
	if err != nil {
		return "", fmt.Errorf("docker ContainerLogs: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("reading container logs: %w", err)
	}
	return string(data), nil
}

// StreamLogs streams container log lines to out, starting from since.
// Blocks until ctx is cancelled or the container exits.
func (c *SDKClient) StreamLogs(ctx context.Context, containerID string, since string, out chan<- string) error {
	logOpts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       "0",
		Since:      since,
	}
	rc, err := c.cli.ContainerLogs(ctx, containerID, logOpts)
	if err != nil {
		return fmt.Errorf("docker ContainerLogs (stream): %w", err)
	}
	defer rc.Close()

	scanner := bufio.NewScanner(rc)
	for scanner.Scan() {
		select {
		case out <- scanner.Text():
		case <-ctx.Done():
			return nil
		}
	}
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		return fmt.Errorf("reading streamed logs: %w", err)
	}
	return nil
}

// ListContainers returns container IDs matching the given label filters.
// Only non-running containers are returned (status: created, exited, dead, paused)
// so that running containers belonging to other services are never removed.
func (c *SDKClient) ListContainers(ctx context.Context, labels map[string]string) ([]string, error) {
	f := filters.NewArgs()
	f.Add("label", "aima.managed=true")
	for k, v := range labels {
		f.Add("label", fmt.Sprintf("%s=%s", k, v))
	}

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true, Filters: f})
	if err != nil {
		return nil, fmt.Errorf("docker ContainerList: %w", err)
	}

	ids := make([]string, 0, len(containers))
	for _, ct := range containers {
		// Only return non-running containers; skip containers that are
		// actively serving another service (status "running" or "restarting").
		if ct.State != "running" && ct.State != "restarting" {
			ids = append(ids, ct.ID)
		}
	}
	return ids, nil
}

// ContainerEvents returns a channel of Docker container events matching filters.
// The channel is closed when ctx is cancelled.
func (c *SDKClient) ContainerEvents(ctx context.Context, filterMap map[string]string) (<-chan ContainerEvent, error) {
	f := filters.NewArgs()
	f.Add("type", "container")
	for k, v := range filterMap {
		f.Add(k, v)
	}

	msgCh, errCh := c.cli.Events(ctx, events.ListOptions{Filters: f})

	ch := make(chan ContainerEvent, 16)
	go func() {
		defer close(ch)
		for {
			select {
			case msg, ok := <-msgCh:
				if !ok {
					return
				}
				ev := ContainerEvent{
					ContainerID: msg.Actor.ID,
					Action:      string(msg.Action),
					Status:      string(msg.Action), // msg.Status is deprecated; Action carries the same info
				}
				select {
				case ch <- ev:
				case <-ctx.Done():
					return
				}
			case err := <-errCh:
				if err != nil && ctx.Err() == nil {
					// Non-cancellation error — close silently; caller checks ctx.
					_ = err
				}
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

// FindContainersByPort returns all containers (regardless of labels) that
// publish the given host port. Uses Docker's "publish" filter so it finds
// externally-created containers that AIMA's label filter would miss.
func (c *SDKClient) FindContainersByPort(ctx context.Context, port int) ([]PortConflict, error) {
	f := filters.NewArgs()
	f.Add("publish", strconv.Itoa(port))

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true, Filters: f})
	if err != nil {
		return nil, fmt.Errorf("docker ContainerList (port filter): %w", err)
	}

	conflicts := make([]PortConflict, 0, len(containers))
	for _, ct := range containers {
		name := ""
		if len(ct.Names) > 0 {
			name = strings.TrimPrefix(ct.Names[0], "/")
		}
		conflicts = append(conflicts, PortConflict{
			ContainerID: ct.ID,
			Name:        name,
			Image:       ct.Image,
			IsAIMA:      ct.Labels["aima.managed"] == "true",
		})
	}
	return conflicts, nil
}

// PullImage pulls a Docker image using the SDK.
func (c *SDKClient) PullImage(ctx context.Context, img string) error {
	rc, err := c.cli.ImagePull(ctx, img, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("docker ImagePull %s: %w", img, err)
	}
	defer rc.Close()
	// Drain the reader to complete the pull; output is JSON progress (discarded).
	_, _ = io.Copy(io.Discard, rc)
	return nil
}

// parseMemory converts strings like "4g", "512m", "1024k" to bytes.
func parseMemory(s string) (int64, error) {
	if len(s) == 0 {
		return 0, fmt.Errorf("empty memory string")
	}
	suffix := s[len(s)-1]
	numStr := s[:len(s)-1]
	num, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid memory %q: %w", s, err)
	}
	switch suffix {
	case 'g', 'G':
		return num * 1024 * 1024 * 1024, nil
	case 'm', 'M':
		return num * 1024 * 1024, nil
	case 'k', 'K':
		return num * 1024, nil
	case 'b', 'B':
		return num, nil
	default:
		// Try treating whole string as bytes.
		return strconv.ParseInt(s, 10, 64)
	}
}

// Compile-time assertion: SDKClient must implement Client.
var _ Client = (*SDKClient)(nil)
