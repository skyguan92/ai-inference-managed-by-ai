package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/config"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestNewStartCommand(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)
	cfg := config.Default()

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
		cfg:      cfg,
	}

	cmd := NewStartCommand(root)
	assert.NotNil(t, cmd)
	assert.Equal(t, "start", cmd.Use)
	assert.NotNil(t, cmd.RunE)
}

func TestStartCommand_Flags(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)
	cfg := config.Default()

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
		cfg:      cfg,
	}

	cmd := NewStartCommand(root)

	portFlag := cmd.Flags().Lookup("port")
	assert.NotNil(t, portFlag)
	assert.Equal(t, "p", portFlag.Shorthand)

	addrFlag := cmd.Flags().Lookup("addr")
	assert.NotNil(t, addrFlag)

	tlsCertFlag := cmd.Flags().Lookup("tls-cert")
	assert.NotNil(t, tlsCertFlag)

	tlsKeyFlag := cmd.Flags().Lookup("tls-key")
	assert.NotNil(t, tlsKeyFlag)
}
