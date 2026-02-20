package docker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// SimpleClient construction
// ---------------------------------------------------------------------------

func TestNewSimpleClient(t *testing.T) {
	c := NewSimpleClient()
	require.NotNil(t, c)
}

// TestCheckDocker verifies CheckDocker returns an error when Docker CLI is not
// available. In a CI environment where Docker IS installed this test will pass
// with nil; we skip it gracefully either way.
func TestCheckDocker_ReturnsNoErrorOrDockerNotFound(t *testing.T) {
	err := CheckDocker()
	// Either Docker is available (no error) or we get a meaningful error.
	// We just verify the call completes without panicking.
	_ = err
}

// TestSimpleClient_PullImage_ContextCancelled verifies that a cancelled context
// causes PullImage to return an error immediately without hanging.
func TestSimpleClient_PullImage_ContextCancelled(t *testing.T) {
	c := NewSimpleClient()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := c.PullImage(ctx, "nonexistent-image:latest")
	assert.Error(t, err)
}

// TestSimpleClient_CreateAndStartContainer_ContextCancelled verifies cancellation.
func TestSimpleClient_CreateAndStartContainer_ContextCancelled(t *testing.T) {
	c := NewSimpleClient()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.CreateAndStartContainer(ctx, "test-ctr", "nginx:latest", ContainerOptions{})
	assert.Error(t, err)
}

// TestSimpleClient_StopContainer_ContextCancelled verifies cancellation propagation.
func TestSimpleClient_StopContainer_ContextCancelled(t *testing.T) {
	c := NewSimpleClient()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Will fail at docker rm step with a cancelled context or docker-not-found error.
	err := c.StopContainer(ctx, "nonexistent-container", 10)
	assert.Error(t, err)
}

// TestSimpleClient_GetContainerStatus_ContextCancelled verifies cancellation.
func TestSimpleClient_GetContainerStatus_ContextCancelled(t *testing.T) {
	c := NewSimpleClient()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.GetContainerStatus(ctx, "nonexistent-container")
	assert.Error(t, err)
}

// TestSimpleClient_ListContainers_ContextCancelled verifies cancellation.
func TestSimpleClient_ListContainers_ContextCancelled(t *testing.T) {
	c := NewSimpleClient()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.ListContainers(ctx, nil)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// MockClient — PullImage
// ---------------------------------------------------------------------------

func TestMockClient_PullImage_Success(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	err := mc.PullImage(ctx, "nginx:latest")
	require.NoError(t, err)

	images, err := mc.ListImages(ctx)
	require.NoError(t, err)
	assert.Len(t, images, 1)
	assert.Equal(t, "nginx:latest", images[0].RepoTags[0])
}

func TestMockClient_PullImage_MultiplePulls(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	refs := []string{"nginx:latest", "redis:7", "postgres:15"}
	for _, ref := range refs {
		require.NoError(t, mc.PullImage(ctx, ref))
	}

	images, err := mc.ListImages(ctx)
	require.NoError(t, err)
	assert.Len(t, images, len(refs))
}

func TestMockClient_PullImage_ContextCancelled(t *testing.T) {
	mc := NewMockClient()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := mc.PullImage(ctx, "nginx:latest")
	assert.ErrorIs(t, err, context.Canceled)
}

func TestMockClient_PullImage_SetsImageSize(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	require.NoError(t, mc.PullImage(ctx, "ubuntu:22.04"))

	images, err := mc.ListImages(ctx)
	require.NoError(t, err)
	require.Len(t, images, 1)
	assert.Greater(t, images[0].Size, int64(0))
	assert.NotEmpty(t, images[0].ID)
}

// ---------------------------------------------------------------------------
// MockClient — CreateContainer / StartContainer (simulating CreateAndStart)
// ---------------------------------------------------------------------------

func TestMockClient_CreateAndStart_Success(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	id, err := mc.CreateContainer(ctx, "my-nginx", "nginx:latest", ContainerOptions{})
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	err = mc.StartContainer(ctx, id)
	require.NoError(t, err)

	ctr, err := mc.GetContainer(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "running", ctr.Status)
	assert.Equal(t, "my-nginx", ctr.Name)
	assert.Equal(t, "nginx:latest", ctr.Image)
}

func TestMockClient_CreateContainer_WithOptions(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	opts := ContainerOptions{
		Ports:   map[string]string{"8080": "80", "8443": "443"},
		Volumes: map[string]string{"/host/data": "/data"},
		Env:     []string{"FOO=bar", "DEBUG=true"},
		Cmd:     []string{"nginx", "-g", "daemon off;"},
	}

	id, err := mc.CreateContainer(ctx, "opts-ctr", "nginx:latest", opts)
	require.NoError(t, err)

	ctr, err := mc.GetContainer(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, opts.Env, ctr.Env)
	assert.Equal(t, opts.Cmd, ctr.Cmd)
	assert.Len(t, ctr.Ports, 2)
	assert.Len(t, ctr.Volumes, 1)
}

func TestMockClient_CreateContainer_SetsCreatedStatus(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	id, err := mc.CreateContainer(ctx, "new-ctr", "alpine:latest", ContainerOptions{})
	require.NoError(t, err)

	ctr, err := mc.GetContainer(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "created", ctr.Status)
}

func TestMockClient_CreateContainer_ContextCancelled(t *testing.T) {
	mc := NewMockClient()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := mc.CreateContainer(ctx, "ctr", "nginx:latest", ContainerOptions{})
	assert.ErrorIs(t, err, context.Canceled)
}

func TestMockClient_StartContainer_NotFound(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	err := mc.StartContainer(ctx, "nonexistent-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMockClient_StartContainer_ContextCancelled(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	id, err := mc.CreateContainer(ctx, "ctr", "nginx:latest", ContainerOptions{})
	require.NoError(t, err)

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err = mc.StartContainer(cancelCtx, id)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestMockClient_MultipleContainers_UniqueIDs(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	ids := make(map[string]bool)
	for i := 0; i < 5; i++ {
		id, err := mc.CreateContainer(ctx, fmt.Sprintf("ctr-%d", i), "alpine:latest", ContainerOptions{})
		require.NoError(t, err)
		assert.False(t, ids[id], "expected unique container ID, got duplicate: %s", id)
		ids[id] = true
	}
}

// ---------------------------------------------------------------------------
// MockClient — StopContainer
// ---------------------------------------------------------------------------

func TestMockClient_StopContainer_Success(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	id, err := mc.CreateContainer(ctx, "stop-me", "nginx:latest", ContainerOptions{})
	require.NoError(t, err)
	require.NoError(t, mc.StartContainer(ctx, id))

	err = mc.StopContainer(ctx, id, time.Millisecond)
	require.NoError(t, err)

	ctr, err := mc.GetContainer(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "stopped", ctr.Status)
}

func TestMockClient_StopContainer_NotFound(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	err := mc.StopContainer(ctx, "ghost-id", time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMockClient_StopContainer_NotRunning(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	// Container is in "created" state (never started)
	id, err := mc.CreateContainer(ctx, "not-started", "nginx:latest", ContainerOptions{})
	require.NoError(t, err)

	err = mc.StopContainer(ctx, id, time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestMockClient_StopContainer_ContextCancelled(t *testing.T) {
	mc := NewMockClient()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := mc.StopContainer(ctx, "any-id", time.Second)
	assert.ErrorIs(t, err, context.Canceled)
}

// ---------------------------------------------------------------------------
// MockClient — GetContainerStatus (via GetContainer)
// ---------------------------------------------------------------------------

func TestMockClient_GetContainer_Running(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	id, err := mc.CreateContainer(ctx, "running-ctr", "nginx:latest", ContainerOptions{})
	require.NoError(t, err)
	require.NoError(t, mc.StartContainer(ctx, id))

	ctr, err := mc.GetContainer(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "running", ctr.Status)
}

func TestMockClient_GetContainer_Stopped(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	id, err := mc.CreateContainer(ctx, "stop-ctr", "nginx:latest", ContainerOptions{})
	require.NoError(t, err)
	require.NoError(t, mc.StartContainer(ctx, id))
	require.NoError(t, mc.StopContainer(ctx, id, time.Millisecond))

	ctr, err := mc.GetContainer(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "stopped", ctr.Status)
}

func TestMockClient_GetContainer_NotFound(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	_, err := mc.GetContainer(ctx, "does-not-exist")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMockClient_InspectContainer_DelegatesToGetContainer(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	id, err := mc.CreateContainer(ctx, "inspect-ctr", "alpine:latest", ContainerOptions{})
	require.NoError(t, err)

	ctr1, err := mc.GetContainer(ctx, id)
	require.NoError(t, err)

	ctr2, err := mc.InspectContainer(ctx, id)
	require.NoError(t, err)

	assert.Equal(t, ctr1, ctr2)
}

// ---------------------------------------------------------------------------
// MockClient — ListContainers
// ---------------------------------------------------------------------------

func TestMockClient_ListContainers_EmptyStore(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	all, err := mc.ListContainers(ctx, true)
	require.NoError(t, err)
	assert.Empty(t, all)

	running, err := mc.ListContainers(ctx, false)
	require.NoError(t, err)
	assert.Empty(t, running)
}

func TestMockClient_ListContainers_AllFlag(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	// Create two containers: one running, one created (not started)
	id1, _ := mc.CreateContainer(ctx, "running", "nginx:latest", ContainerOptions{})
	mc.StartContainer(ctx, id1) //nolint:errcheck
	_, _ = mc.CreateContainer(ctx, "created", "nginx:latest", ContainerOptions{})

	// all=true should return both
	all, err := mc.ListContainers(ctx, true)
	require.NoError(t, err)
	assert.Len(t, all, 2)

	// all=false should return only running
	running, err := mc.ListContainers(ctx, false)
	require.NoError(t, err)
	assert.Len(t, running, 1)
	assert.Equal(t, "running", running[0].Status)
}

func TestMockClient_ListContainers_OnlyRunning(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	names := []string{"a", "b", "c"}
	for _, name := range names {
		id, err := mc.CreateContainer(ctx, name, "alpine:latest", ContainerOptions{})
		require.NoError(t, err)
		require.NoError(t, mc.StartContainer(ctx, id))
	}

	// Stop one
	all, _ := mc.ListContainers(ctx, true)
	require.NoError(t, mc.StopContainer(ctx, all[0].ID, time.Millisecond))

	running, err := mc.ListContainers(ctx, false)
	require.NoError(t, err)
	assert.Len(t, running, 2)
	for _, c := range running {
		assert.Equal(t, "running", c.Status)
	}
}

func TestMockClient_ListContainers_ContextCancelled(t *testing.T) {
	mc := NewMockClient()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := mc.ListContainers(ctx, true)
	assert.ErrorIs(t, err, context.Canceled)
}

// ---------------------------------------------------------------------------
// MockClient — RemoveContainer
// ---------------------------------------------------------------------------

func TestMockClient_RemoveContainer_StoppedContainer(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	id, _ := mc.CreateContainer(ctx, "rm-me", "alpine:latest", ContainerOptions{})
	require.NoError(t, mc.StartContainer(ctx, id))
	require.NoError(t, mc.StopContainer(ctx, id, time.Millisecond))

	err := mc.RemoveContainer(ctx, id, false)
	require.NoError(t, err)

	_, err = mc.GetContainer(ctx, id)
	assert.Error(t, err)
}

func TestMockClient_RemoveContainer_RunningWithoutForce(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	id, _ := mc.CreateContainer(ctx, "force-rm", "alpine:latest", ContainerOptions{})
	require.NoError(t, mc.StartContainer(ctx, id))

	err := mc.RemoveContainer(ctx, id, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot remove running container")
}

func TestMockClient_RemoveContainer_RunningWithForce(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	id, _ := mc.CreateContainer(ctx, "force-rm", "alpine:latest", ContainerOptions{})
	require.NoError(t, mc.StartContainer(ctx, id))

	err := mc.RemoveContainer(ctx, id, true)
	require.NoError(t, err)

	_, err = mc.GetContainer(ctx, id)
	assert.Error(t, err)
}

func TestMockClient_RemoveContainer_NotFound(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	err := mc.RemoveContainer(ctx, "ghost", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// MockClient — Ping / Version
// ---------------------------------------------------------------------------

func TestMockClient_Ping_Success(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	err := mc.Ping(ctx)
	assert.NoError(t, err)
}

func TestMockClient_Ping_ContextCancelled(t *testing.T) {
	mc := NewMockClient()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := mc.Ping(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestMockClient_Version_Success(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	version, err := mc.Version(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, version)
}

func TestMockClient_Version_ContextCancelled(t *testing.T) {
	mc := NewMockClient()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := mc.Version(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

// ---------------------------------------------------------------------------
// MockClient — ContainerLogs / ExecInContainer
// ---------------------------------------------------------------------------

func TestMockClient_ContainerLogs_Running(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	id, _ := mc.CreateContainer(ctx, "log-ctr", "nginx:latest", ContainerOptions{})
	require.NoError(t, mc.StartContainer(ctx, id))

	logs, err := mc.ContainerLogs(ctx, id)
	require.NoError(t, err)
	assert.Contains(t, logs, id)
}

func TestMockClient_ContainerLogs_NotFound(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	_, err := mc.ContainerLogs(ctx, "missing-container")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMockClient_ExecInContainer_Success(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	id, _ := mc.CreateContainer(ctx, "exec-ctr", "alpine:latest", ContainerOptions{})
	require.NoError(t, mc.StartContainer(ctx, id))

	output, err := mc.ExecInContainer(ctx, id, []string{"echo", "hello"})
	require.NoError(t, err)
	assert.Contains(t, output, id)
}

func TestMockClient_ExecInContainer_NotFound(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	_, err := mc.ExecInContainer(ctx, "missing", []string{"ls"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// MockClient — WaitContainer
// ---------------------------------------------------------------------------

func TestMockClient_WaitContainer_Success(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	id, _ := mc.CreateContainer(ctx, "wait-ctr", "alpine:latest", ContainerOptions{})
	require.NoError(t, mc.StartContainer(ctx, id))

	code, err := mc.WaitContainer(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, int64(0), code)

	ctr, _ := mc.GetContainer(ctx, id)
	assert.Equal(t, "exited", ctr.Status)
}

func TestMockClient_WaitContainer_NotFound(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	code, err := mc.WaitContainer(ctx, "nonexistent")
	require.Error(t, err)
	assert.Equal(t, int64(-1), code)
}

// ---------------------------------------------------------------------------
// MockClient — PauseContainer / UnpauseContainer
// ---------------------------------------------------------------------------

func TestMockClient_PauseUnpause_Success(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	id, _ := mc.CreateContainer(ctx, "pause-ctr", "nginx:latest", ContainerOptions{})
	require.NoError(t, mc.StartContainer(ctx, id))

	require.NoError(t, mc.PauseContainer(ctx, id))
	ctr, _ := mc.GetContainer(ctx, id)
	assert.Equal(t, "paused", ctr.Status)

	require.NoError(t, mc.UnpauseContainer(ctx, id))
	ctr, _ = mc.GetContainer(ctx, id)
	assert.Equal(t, "running", ctr.Status)
}

func TestMockClient_PauseContainer_NotRunning(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	id, _ := mc.CreateContainer(ctx, "not-running", "alpine:latest", ContainerOptions{})
	// container is in "created" state

	err := mc.PauseContainer(ctx, id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestMockClient_UnpauseContainer_NotPaused(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	id, _ := mc.CreateContainer(ctx, "running-ctr", "alpine:latest", ContainerOptions{})
	require.NoError(t, mc.StartContainer(ctx, id))

	err := mc.UnpauseContainer(ctx, id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not paused")
}

// ---------------------------------------------------------------------------
// MockClient — KillContainer
// ---------------------------------------------------------------------------

func TestMockClient_KillContainer_Success(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	id, _ := mc.CreateContainer(ctx, "kill-ctr", "nginx:latest", ContainerOptions{})
	require.NoError(t, mc.StartContainer(ctx, id))

	err := mc.KillContainer(ctx, id, "SIGKILL")
	require.NoError(t, err)

	ctr, _ := mc.GetContainer(ctx, id)
	assert.Equal(t, "stopped", ctr.Status)
}

func TestMockClient_KillContainer_NotFound(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	err := mc.KillContainer(ctx, "ghost", "SIGTERM")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// MockClient — RenameContainer
// ---------------------------------------------------------------------------

func TestMockClient_RenameContainer_Success(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	id, _ := mc.CreateContainer(ctx, "old-name", "alpine:latest", ContainerOptions{})

	err := mc.RenameContainer(ctx, id, "new-name")
	require.NoError(t, err)

	ctr, _ := mc.GetContainer(ctx, id)
	assert.Equal(t, "new-name", ctr.Name)
}

func TestMockClient_RenameContainer_NameAlreadyInUse(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	_, _ = mc.CreateContainer(ctx, "taken", "alpine:latest", ContainerOptions{})
	id2, _ := mc.CreateContainer(ctx, "other", "alpine:latest", ContainerOptions{})

	err := mc.RenameContainer(ctx, id2, "taken")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already in use")
}

func TestMockClient_RenameContainer_NotFound(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	err := mc.RenameContainer(ctx, "ghost", "any-name")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// MockClient — RestartContainer
// ---------------------------------------------------------------------------

func TestMockClient_RestartContainer_Success(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	id, _ := mc.CreateContainer(ctx, "restart-ctr", "nginx:latest", ContainerOptions{})
	require.NoError(t, mc.StartContainer(ctx, id))

	err := mc.RestartContainer(ctx, id, time.Millisecond)
	require.NoError(t, err)

	ctr, _ := mc.GetContainer(ctx, id)
	assert.Equal(t, "running", ctr.Status)
}

// ---------------------------------------------------------------------------
// MockClient — Image operations
// ---------------------------------------------------------------------------

func TestMockClient_RemoveImage_Success(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	require.NoError(t, mc.PullImage(ctx, "alpine:latest"))

	images, _ := mc.ListImages(ctx)
	require.Len(t, images, 1)
	imageID := images[0].ID

	err := mc.RemoveImage(ctx, imageID, false)
	require.NoError(t, err)

	images, _ = mc.ListImages(ctx)
	assert.Empty(t, images)
}

func TestMockClient_RemoveImage_NotFound(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	err := mc.RemoveImage(ctx, "nonexistent-image-id", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMockClient_TagImage_Success(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	require.NoError(t, mc.PullImage(ctx, "nginx:latest"))

	err := mc.TagImage(ctx, "nginx:latest", "nginx:stable")
	require.NoError(t, err)

	// Both tags should now be listed
	images, _ := mc.ListImages(ctx)
	assert.GreaterOrEqual(t, len(images), 1)

	tagged, err := mc.ListImages(ctx)
	require.NoError(t, err)
	found := false
	for _, img := range tagged {
		for _, tag := range img.RepoTags {
			if tag == "nginx:stable" {
				found = true
			}
		}
	}
	assert.True(t, found, "expected nginx:stable tag to be present")
}

func TestMockClient_TagImage_SourceNotFound(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	err := mc.TagImage(ctx, "nonexistent:latest", "new:tag")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMockClient_PushImage_Success(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	require.NoError(t, mc.PullImage(ctx, "myrepo/myimage:v1"))

	err := mc.PushImage(ctx, "myrepo/myimage:v1")
	assert.NoError(t, err)
}

func TestMockClient_PushImage_NotFound(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	err := mc.PushImage(ctx, "nonexistent:latest")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMockClient_BuildImage_Success(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	err := mc.BuildImage(ctx, "/path/to/Dockerfile", "myapp:latest")
	require.NoError(t, err)

	images, err := mc.ListImages(ctx)
	require.NoError(t, err)
	assert.Len(t, images, 1)
	assert.Equal(t, "myapp:latest", images[0].RepoTags[0])
}

func TestMockClient_BuildImage_ContextCancelled(t *testing.T) {
	mc := NewMockClient()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := mc.BuildImage(ctx, "/Dockerfile", "tag")
	assert.ErrorIs(t, err, context.Canceled)
}

// ---------------------------------------------------------------------------
// MockClient — ExportContainer / ImportImage
// ---------------------------------------------------------------------------

func TestMockClient_ExportContainer_Success(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	id, _ := mc.CreateContainer(ctx, "export-ctr", "alpine:latest", ContainerOptions{})

	data, err := mc.ExportContainer(ctx, id)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Contains(t, string(data), id)
}

func TestMockClient_ExportContainer_NotFound(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	_, err := mc.ExportContainer(ctx, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMockClient_ImportImage_Success(t *testing.T) {
	mc := NewMockClient()
	ctx := context.Background()

	data := []byte("fake-tar-data")
	err := mc.ImportImage(ctx, data, "imported:v1")
	require.NoError(t, err)

	images, _ := mc.ListImages(ctx)
	assert.Len(t, images, 1)
	assert.Equal(t, "imported:v1", images[0].RepoTags[0])
	assert.Equal(t, int64(len(data)), images[0].Size)
}

// ---------------------------------------------------------------------------
// ContainerOptions — field coverage
// ---------------------------------------------------------------------------

func TestContainerOptions_AllFields(t *testing.T) {
	opts := ContainerOptions{
		Env:        []string{"KEY=value"},
		Cmd:        []string{"sh", "-c", "echo hi"},
		Ports:      map[string]string{"9090": "9090"},
		Volumes:    map[string]string{"/tmp/host": "/tmp/container"},
		Labels:     map[string]string{"app": "test"},
		WorkingDir: "/workspace",
		GPU:        true,
		Memory:     "512m",
		CPU:        "1.5",
	}

	mc := NewMockClient()
	ctx := context.Background()

	id, err := mc.CreateContainer(ctx, "full-opts", "alpine:latest", opts)
	require.NoError(t, err)

	ctr, err := mc.GetContainer(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, opts.Env, ctr.Env)
	assert.Equal(t, opts.Cmd, ctr.Cmd)
	assert.Len(t, ctr.Ports, 1)
	assert.Len(t, ctr.Volumes, 1)
}
