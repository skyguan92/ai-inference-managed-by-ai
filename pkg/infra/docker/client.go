package docker

import "context"

// ContainerEvent represents a Docker container lifecycle event.
type ContainerEvent struct {
	// ContainerID is the ID of the container that generated the event.
	ContainerID string
	// Action is the event action, e.g. "start", "stop", "die", "kill".
	Action string
	// Status is the event status, typically the same as Action.
	Status string
}

// Client is the interface for Docker container lifecycle and image operations.
type Client interface {
	// PullImage pulls a Docker image.
	PullImage(ctx context.Context, image string) error

	// CreateAndStartContainer creates and starts a container, returning its ID.
	CreateAndStartContainer(ctx context.Context, name, image string, opts ContainerOptions) (string, error)

	// StopContainer stops and removes a container. timeout is in seconds.
	StopContainer(ctx context.Context, containerID string, timeout int) error

	// GetContainerStatus returns the container state (e.g. "running", "exited").
	GetContainerStatus(ctx context.Context, containerID string) (string, error)

	// GetContainerLogs returns the last `tail` lines of container logs.
	GetContainerLogs(ctx context.Context, containerID string, tail int) (string, error)

	// StreamLogs streams container log lines written since `since` to `out`.
	// The function blocks until ctx is cancelled or the container exits.
	StreamLogs(ctx context.Context, containerID string, since string, out chan<- string) error

	// ListContainers returns container IDs matching the given label filters.
	ListContainers(ctx context.Context, labels map[string]string) ([]string, error)

	// ContainerEvents returns a channel of container events matching filters.
	// The channel is closed when ctx is cancelled.
	ContainerEvents(ctx context.Context, filters map[string]string) (<-chan ContainerEvent, error)
}

// Compile-time assertion: SimpleClient must implement Client.
var _ Client = (*SimpleClient)(nil)
