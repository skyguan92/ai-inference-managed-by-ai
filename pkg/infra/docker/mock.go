package docker

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ContainerOptions 创建容器的选项
type ContainerOptions struct {
	Env        []string
	Cmd        []string
	Ports      map[string]string
	Volumes    map[string]string
	Labels     map[string]string
	WorkingDir string
	GPU        bool
	Memory     string // e.g., "4g", "512m"
	CPU        string // e.g., "2.0"
}

// MockClient 用于测试的 Mock Docker 客户端
type MockClient struct {
	Containers map[string]*MockContainer
	Images     map[string]*MockImage
}

// MockContainer 模拟容器
type MockContainer struct {
	ID      string
	Name    string
	Image   string
	Status  string
	Env     []string
	Cmd     []string
	Ports   []string
	Volumes []string
	Labels  map[string]string
}

// MockImage 模拟镜像
type MockImage struct {
	ID       string
	RepoTags []string
	Size     int64
}

// NewMockClient 创建新的 Mock Docker 客户端
func NewMockClient() *MockClient {
	return &MockClient{
		Containers: make(map[string]*MockContainer),
		Images:     make(map[string]*MockImage),
	}
}

// CreateContainer 创建容器
func (c *MockClient) CreateContainer(ctx context.Context, name, image string, opts ContainerOptions) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	containerID := fmt.Sprintf("mock-container-%d", len(c.Containers)+1)

	ports := make([]string, 0)
	for hostPort, containerPort := range opts.Ports {
		ports = append(ports, fmt.Sprintf("%s:%s", hostPort, containerPort))
	}

	volumes := make([]string, 0)
	for hostPath, containerPath := range opts.Volumes {
		volumes = append(volumes, fmt.Sprintf("%s:%s", hostPath, containerPath))
	}

	labels := make(map[string]string, len(opts.Labels))
	for k, v := range opts.Labels {
		labels[k] = v
	}

	container := &MockContainer{
		ID:      containerID,
		Name:    name,
		Image:   image,
		Status:  "created",
		Env:     opts.Env,
		Cmd:     opts.Cmd,
		Ports:   ports,
		Volumes: volumes,
		Labels:  labels,
	}

	c.Containers[containerID] = container
	return containerID, nil
}

// StartContainer 启动容器
func (c *MockClient) StartContainer(ctx context.Context, containerID string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	container, exists := c.Containers[containerID]
	if !exists {
		return fmt.Errorf("container %s not found", containerID)
	}

	container.Status = "running"
	return nil
}

// StopContainer implements docker.Client: stops and removes a container. timeout is in seconds.
func (c *MockClient) StopContainer(ctx context.Context, containerID string, timeout int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	container, exists := c.Containers[containerID]
	if !exists {
		return nil // idempotent — already gone
	}

	container.Status = "stopped"
	delete(c.Containers, containerID)
	return nil
}

// RemoveContainer 删除容器
func (c *MockClient) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	container, exists := c.Containers[containerID]
	if !exists {
		return fmt.Errorf("container %s not found", containerID)
	}

	if container.Status == "running" && !force {
		return fmt.Errorf("cannot remove running container %s", containerID)
	}

	delete(c.Containers, containerID)
	return nil
}

// ListAllContainers lists all containers (optionally including stopped ones).
// Use ListContainers (docker.Client interface) for label-filtered listing.
func (c *MockClient) ListAllContainers(ctx context.Context, all bool) ([]*MockContainer, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	containers := make([]*MockContainer, 0, len(c.Containers))
	for _, container := range c.Containers {
		if !all && container.Status != "running" {
			continue
		}
		containers = append(containers, container)
	}
	return containers, nil
}

// GetContainer 获取容器信息
func (c *MockClient) GetContainer(ctx context.Context, containerID string) (*MockContainer, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	container, exists := c.Containers[containerID]
	if !exists {
		return nil, fmt.Errorf("container %s not found", containerID)
	}
	return container, nil
}

// PullImage 拉取镜像
func (c *MockClient) PullImage(ctx context.Context, imageRef string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	imageID := fmt.Sprintf("mock-image-%d", len(c.Images)+1)
	c.Images[imageRef] = &MockImage{
		ID:       imageID,
		RepoTags: []string{imageRef},
		Size:     1024 * 1024 * 100, // 100MB
	}
	return nil
}

// ListImages 列出所有镜像
func (c *MockClient) ListImages(ctx context.Context) ([]*MockImage, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	images := make([]*MockImage, 0, len(c.Images))
	for _, image := range c.Images {
		images = append(images, image)
	}
	return images, nil
}

// RemoveImage 删除镜像
func (c *MockClient) RemoveImage(ctx context.Context, imageID string, force bool) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	for ref, image := range c.Images {
		if image.ID == imageID {
			delete(c.Images, ref)
			return nil
		}
	}
	return fmt.Errorf("image %s not found", imageID)
}

// ContainerLogs 获取容器日志
func (c *MockClient) ContainerLogs(ctx context.Context, containerID string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	_, exists := c.Containers[containerID]
	if !exists {
		return "", fmt.Errorf("container %s not found", containerID)
	}

	return fmt.Sprintf("Mock logs for container %s", containerID), nil
}

// ExecInContainer 在容器中执行命令
func (c *MockClient) ExecInContainer(ctx context.Context, containerID string, cmd []string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	_, exists := c.Containers[containerID]
	if !exists {
		return "", fmt.Errorf("container %s not found", containerID)
	}

	return fmt.Sprintf("Executed %v in container %s", cmd, containerID), nil
}

// InspectContainer 检查容器详情
func (c *MockClient) InspectContainer(ctx context.Context, containerID string) (*MockContainer, error) {
	return c.GetContainer(ctx, containerID)
}

// WaitContainer 等待容器结束
func (c *MockClient) WaitContainer(ctx context.Context, containerID string) (int64, error) {
	select {
	case <-ctx.Done():
		return -1, ctx.Err()
	default:
	}

	container, exists := c.Containers[containerID]
	if !exists {
		return -1, fmt.Errorf("container %s not found", containerID)
	}

	container.Status = "exited"
	return 0, nil
}

// RestartContainer 重启容器
func (c *MockClient) RestartContainer(ctx context.Context, containerID string, timeout time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	container, exists := c.Containers[containerID]
	if !exists {
		return fmt.Errorf("container %s not found", containerID)
	}
	container.Status = "running"
	return nil
}

// PauseContainer 暂停容器
func (c *MockClient) PauseContainer(ctx context.Context, containerID string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	container, exists := c.Containers[containerID]
	if !exists {
		return fmt.Errorf("container %s not found", containerID)
	}

	if container.Status != "running" {
		return fmt.Errorf("container %s is not running", containerID)
	}

	container.Status = "paused"
	return nil
}

// UnpauseContainer 恢复容器
func (c *MockClient) UnpauseContainer(ctx context.Context, containerID string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	container, exists := c.Containers[containerID]
	if !exists {
		return fmt.Errorf("container %s not found", containerID)
	}

	if container.Status != "paused" {
		return fmt.Errorf("container %s is not paused", containerID)
	}

	container.Status = "running"
	return nil
}

// KillContainer 强制终止容器
func (c *MockClient) KillContainer(ctx context.Context, containerID string, signal string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	container, exists := c.Containers[containerID]
	if !exists {
		return fmt.Errorf("container %s not found", containerID)
	}

	container.Status = "stopped"
	return nil
}

// RenameContainer 重命名容器
func (c *MockClient) RenameContainer(ctx context.Context, containerID, newName string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	container, exists := c.Containers[containerID]
	if !exists {
		return fmt.Errorf("container %s not found", containerID)
	}

	// 检查新名称是否已被使用
	for _, c := range c.Containers {
		if c.Name == newName {
			return fmt.Errorf("container name %s already in use", newName)
		}
	}

	container.Name = newName
	return nil
}

// Ping 检查 Docker 守护进程是否可达
func (c *MockClient) Ping(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return nil
}

// Version 获取 Docker 版本信息
func (c *MockClient) Version(ctx context.Context) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	return "mock-docker-1.0.0", nil
}

// BuildImage 构建镜像
func (c *MockClient) BuildImage(ctx context.Context, dockerfilePath, tag string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	imageID := fmt.Sprintf("mock-built-image-%d", len(c.Images)+1)
	c.Images[tag] = &MockImage{
		ID:       imageID,
		RepoTags: []string{tag},
		Size:     1024 * 1024 * 50, // 50MB
	}
	return nil
}

// TagImage 给镜像打标签
func (c *MockClient) TagImage(ctx context.Context, sourceImage, targetTag string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	image, exists := c.Images[sourceImage]
	if !exists {
		return fmt.Errorf("image %s not found", sourceImage)
	}

	image.RepoTags = append(image.RepoTags, targetTag)
	c.Images[targetTag] = image
	return nil
}

// PushImage 推送镜像
func (c *MockClient) PushImage(ctx context.Context, imageRef string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	_, exists := c.Images[imageRef]
	if !exists {
		return fmt.Errorf("image %s not found", imageRef)
	}
	return nil
}

// ExportContainer 导出容器为 tar 归档
func (c *MockClient) ExportContainer(ctx context.Context, containerID string) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	_, exists := c.Containers[containerID]
	if !exists {
		return nil, fmt.Errorf("container %s not found", containerID)
	}

	// 返回模拟的 tar 数据
	return []byte(fmt.Sprintf("mock-tar-data-for-%s", containerID)), nil
}

// ImportImage 从 tar 归档导入镜像
func (c *MockClient) ImportImage(ctx context.Context, source []byte, tag string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	imageID := fmt.Sprintf("mock-imported-image-%d", len(c.Images)+1)
	c.Images[tag] = &MockImage{
		ID:       imageID,
		RepoTags: []string{tag},
		Size:     int64(len(source)),
	}
	return nil
}

// ---- docker.Client interface implementation ----

// CreateAndStartContainer implements docker.Client: creates and starts a container atomically.
func (c *MockClient) CreateAndStartContainer(ctx context.Context, name, image string, opts ContainerOptions) (string, error) {
	containerID, err := c.CreateContainer(ctx, name, image, opts)
	if err != nil {
		return "", err
	}
	if err := c.StartContainer(ctx, containerID); err != nil {
		return "", err
	}
	return containerID, nil
}

// GetContainerStatus implements docker.Client: returns the container status string.
func (c *MockClient) GetContainerStatus(ctx context.Context, containerID string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	container, exists := c.Containers[containerID]
	if !exists {
		return "", fmt.Errorf("container %s not found", containerID)
	}
	return container.Status, nil
}

// GetContainerLogs implements docker.Client: returns the last tail lines of container logs.
func (c *MockClient) GetContainerLogs(ctx context.Context, containerID string, tail int) (string, error) {
	logs, err := c.ContainerLogs(ctx, containerID)
	if err != nil {
		return "", err
	}
	_ = strconv.Itoa(tail) // tail ignored in mock; all logs returned
	return logs, nil
}

// StreamLogs implements docker.Client: sends a single mock log line and returns.
func (c *MockClient) StreamLogs(ctx context.Context, containerID string, since string, out chan<- string) error {
	select {
	case <-ctx.Done():
		return nil
	default:
	}
	_, exists := c.Containers[containerID]
	if !exists {
		return fmt.Errorf("container %s not found", containerID)
	}
	select {
	case out <- fmt.Sprintf("Mock log line for container %s", containerID):
	case <-ctx.Done():
	}
	return nil
}

// ListContainers implements docker.Client: returns container IDs matching label filters.
// Includes containers in all states (running, created, exited, etc.) to match the
// real client behavior, which uses All: true / -a flag.
func (c *MockClient) ListContainers(ctx context.Context, labels map[string]string) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	var ids []string
	for id := range c.Containers {
		ids = append(ids, id)
	}
	return ids, nil
}

// ContainerEvents implements docker.Client: returns a channel that is immediately closed (no events in mock).
func (c *MockClient) ContainerEvents(ctx context.Context, filters map[string]string) (<-chan ContainerEvent, error) {
	ch := make(chan ContainerEvent)
	close(ch)
	return ch, nil
}

// FindContainersByPort implements docker.Client: returns containers publishing the given host port.
func (c *MockClient) FindContainersByPort(ctx context.Context, port int) ([]PortConflict, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	portStr := strconv.Itoa(port)
	var conflicts []PortConflict
	for _, ct := range c.Containers {
		for _, p := range ct.Ports {
			// Ports are stored as "hostPort:containerPort" strings.
			parts := strings.SplitN(p, ":", 2)
			if parts[0] == portStr {
				isAIMA := ct.Labels["aima.managed"] == "true"
				conflicts = append(conflicts, PortConflict{
					ContainerID: ct.ID,
					Name:        ct.Name,
					Image:       ct.Image,
					IsAIMA:      isAIMA,
				})
				break
			}
		}
	}
	return conflicts, nil
}

// Compile-time assertion: MockClient must implement docker.Client.
var _ Client = (*MockClient)(nil)
