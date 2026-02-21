package docker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockClient_FindContainersByPort_Empty(t *testing.T) {
	c := NewMockClient()
	conflicts, err := c.FindContainersByPort(context.Background(), 8000)
	require.NoError(t, err)
	assert.Empty(t, conflicts)
}

func TestMockClient_FindContainersByPort_AIMAContainer(t *testing.T) {
	c := NewMockClient()
	// Seed a container that is AIMA-managed and publishes port 8000.
	c.Containers["ctr-1"] = &MockContainer{
		ID:     "ctr-1",
		Name:   "aima-vllm-123",
		Image:  "vllm:latest",
		Status: "running",
		Ports:  []string{"8000:8000"},
		Labels: map[string]string{"aima.managed": "true", "aima.engine": "vllm"},
	}

	conflicts, err := c.FindContainersByPort(context.Background(), 8000)
	require.NoError(t, err)
	require.Len(t, conflicts, 1)
	assert.Equal(t, "ctr-1", conflicts[0].ContainerID)
	assert.Equal(t, "aima-vllm-123", conflicts[0].Name)
	assert.Equal(t, "vllm:latest", conflicts[0].Image)
	assert.True(t, conflicts[0].IsAIMA)
}

func TestMockClient_FindContainersByPort_NonAIMAContainer(t *testing.T) {
	c := NewMockClient()
	// Seed an externally-created container with no AIMA labels.
	c.Containers["ext-1"] = &MockContainer{
		ID:     "ext-1",
		Name:   "nginx",
		Image:  "nginx:latest",
		Status: "running",
		Ports:  []string{"8000:80"},
		Labels: map[string]string{},
	}

	conflicts, err := c.FindContainersByPort(context.Background(), 8000)
	require.NoError(t, err)
	require.Len(t, conflicts, 1)
	assert.Equal(t, "ext-1", conflicts[0].ContainerID)
	assert.False(t, conflicts[0].IsAIMA)
}

func TestMockClient_FindContainersByPort_MultipleContainers(t *testing.T) {
	c := NewMockClient()
	c.Containers["ctr-a"] = &MockContainer{
		ID:     "ctr-a",
		Name:   "aima-vllm",
		Image:  "vllm:latest",
		Status: "running",
		Ports:  []string{"8000:8000"},
		Labels: map[string]string{"aima.managed": "true"},
	}
	c.Containers["ctr-b"] = &MockContainer{
		ID:     "ctr-b",
		Name:   "busybox",
		Image:  "busybox",
		Status: "running",
		Ports:  []string{"8000:8000"},
		Labels: map[string]string{},
	}
	// Container on a different port — should NOT appear.
	c.Containers["ctr-c"] = &MockContainer{
		ID:     "ctr-c",
		Name:   "other",
		Image:  "other:latest",
		Status: "running",
		Ports:  []string{"9000:9000"},
		Labels: map[string]string{"aima.managed": "true"},
	}

	conflicts, err := c.FindContainersByPort(context.Background(), 8000)
	require.NoError(t, err)
	assert.Len(t, conflicts, 2)

	byID := make(map[string]PortConflict)
	for _, cf := range conflicts {
		byID[cf.ContainerID] = cf
	}
	assert.True(t, byID["ctr-a"].IsAIMA)
	assert.False(t, byID["ctr-b"].IsAIMA)
}

func TestMockClient_FindContainersByPort_ContextCancelled(t *testing.T) {
	c := NewMockClient()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	_, err := c.FindContainersByPort(ctx, 8000)
	assert.ErrorIs(t, err, context.Canceled)
}
