package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	coreagent "github.com/jguan/ai-inference-managed-by-ai/pkg/agent"
	agentllm "github.com/jguan/ai-inference-managed-by-ai/pkg/agent/llm"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/config"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/eventbus"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/hal/nvidia"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/provider"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/provider/huggingface"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/store"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/registry"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/engine"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/service"
)

var (
	cliVersion   = "dev"
	cliBuildDate = "unknown"
	cliGitCommit = "unknown"
)

type RootCommand struct {
	cmd          *cobra.Command
	cfg          *config.Config
	gateway      *gateway.Gateway
	registry     *unit.Registry
	serviceStore service.ServiceStore
	eventBus     eventbus.EventBus
	opts         *OutputOptions
	formatStr    string
}

func NewRootCommand() *RootCommand {
	root := &RootCommand{
		opts: NewOutputOptions(),
	}

	cmd := &cobra.Command{
		Use:   "aima",
		Short: "AIMA - AI Inference Managed by AI",
		Long: `AIMA (AI Inference Managed by AI) is a comprehensive 
AI inference infrastructure management platform.

It provides a unified interface for managing models, inference engines,
hardware devices, and resources through HTTP, MCP, and CLI.`,
		PersistentPreRunE: root.persistentPreRunE,
	}

	pflags := cmd.PersistentFlags()

	pflags.StringVarP(&root.formatStr, "output", "o", "table", "Output format (table, json, yaml)")
	pflags.BoolVarP(&root.opts.Quiet, "quiet", "q", false, "Suppress output")
	pflags.String("config", "", "Config file path (default: ~/.aima/config.yaml)")

	_ = viper.BindPFlag("output", pflags.Lookup("output"))
	_ = viper.BindPFlag("quiet", pflags.Lookup("quiet"))
	_ = viper.BindPFlag("config", pflags.Lookup("config"))

	root.cmd = cmd

	root.addSubCommands()

	return root
}

func (r *RootCommand) persistentPreRunE(cmd *cobra.Command, args []string) error {
	r.opts.Format = OutputFormat(r.formatStr)

	cfgPath := viper.GetString("config")
	var err error
	r.cfg, err = config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	r.registry = unit.NewRegistry()

	// Create data directory if not exists
	dataDir := r.cfg.General.DataDir
	if dataDir == "" {
		dataDir = "~/.aima"
	}
	// Expand home directory
	if dataDir[:2] == "~/" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, dataDir[2:])
	}

	// Create SQLite store for models and services
	dbPath := filepath.Join(dataDir, "aima.db")
	var modelStore model.ModelStore
	var serviceStore service.ServiceStore

	sqliteStore, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		// Fallback to file store if SQLite fails
		slog.Warn("failed to create SQLite store, trying file store", "error", err)
		fileStore, err := store.NewFileStore(dataDir)
		if err != nil {
			slog.Warn("failed to create file store, using memory store", "error", err)
			modelStore = model.NewMemoryStore()
			serviceStore = service.NewMemoryStore()
		} else {
			modelStore = fileStore
			serviceStore = service.NewMemoryStore()
		}
	} else {
		slog.Info("using SQLite database for persistent storage")
		modelStore = sqliteStore
		// Create service store using the same database
		svcStore, err := store.NewServiceSQLiteStore(sqliteStore.DB())
		if err != nil {
			slog.Warn("failed to create service SQLite store, using memory store", "error", err)
			serviceStore = service.NewMemoryStore()
		} else {
			serviceStore = svcStore
		}
	}

	// Create providers
	modelProvider := huggingface.NewProvider(
		huggingface.WithDownloadDir(r.cfg.Model.StorageDir),
	)

	// Create hybrid engine provider (supports Docker + Native modes)
	slog.Info("initializing hybrid engine provider", "mode", "Docker + Native")
	serviceProvider := provider.NewHybridServiceProvider(modelStore, serviceStore)
	engineProvider := serviceProvider.GetEngineProvider()

	// Create event bus and wire it to the engine provider for progress events
	bus := eventbus.NewInMemoryEventBus()
	r.eventBus = bus
	if hep, ok := engineProvider.(*provider.HybridEngineProvider); ok {
		hep.SetEventBus(bus)
	}

	// Create engine store (memory-based for now)
	engineStore := engine.NewMemoryStore()

	// Seed engine store with available engine types from loaded assets.
	if hep, ok := engineProvider.(*provider.HybridEngineProvider); ok {
		for _, assetType := range hep.AssetTypes() {
			now := time.Now().Unix()
			if err := engineStore.Create(context.Background(), &engine.Engine{
				ID:        "engine-" + uuid.New().String()[:8],
				Name:      assetType,
				Type:      engine.EngineType(assetType),
				Status:    engine.EngineStatusStopped,
				CreatedAt: now,
				UpdatedAt: now,
			}); err != nil {
				slog.Warn("failed to seed engine store", "type", assetType, "error", err)
			}
		}
	}

	// Create device provider (NVIDIA GPU detection via nvidia-smi)
	deviceProvider := nvidia.NewProvider()

	// Expose serviceStore to setupAgent for local LLM auto-detection
	r.serviceStore = serviceStore

	// Create inference provider that proxies to running services
	inferenceProvider := provider.NewProxyInferenceProvider(serviceStore, modelStore)

	// Register all atomic units with providers
	if err := registry.RegisterAll(r.registry,
		registry.WithModelProvider(modelProvider),
		registry.WithModelStore(modelStore),
		registry.WithServiceProvider(serviceProvider),
		registry.WithServiceStore(serviceStore),
		registry.WithEngineProvider(engineProvider),
		registry.WithEngineStore(engineStore),
		registry.WithDeviceProvider(deviceProvider),
		registry.WithInferenceProvider(inferenceProvider),
	); err != nil {
		return fmt.Errorf("register units: %w", err)
	}

	r.gateway = gateway.NewGateway(r.registry, gateway.WithTimeout(r.cfg.Gateway.RequestTimeoutD))

	// Two-phase agent setup: create Agent after Gateway so MCPAdapter can be used
	// as the ToolExecutor (it needs the Gateway to dispatch tool calls).
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if err := r.setupAgent(ctx); err != nil {
		slog.Warn("agent setup failed, agent commands will be unavailable", "error", err)
	}

	return nil
}

// setupAgent creates the LLM client, wraps the MCPAdapter as a ToolExecutor,
// constructs the Agent, and registers the agent domain with the registry.
//
// When no API key is configured, it tries to auto-detect a locally running
// AIMA inference service (via serviceStore) or a local Ollama instance.
// Returns nil without registering an agent when no LLM backend is available.
func (r *RootCommand) setupAgent(ctx context.Context) error {
	cfg := r.cfg.Agent

	var llmClient agentllm.LLMClient

	if cfg.LLMAPIKey == "" {
		// No cloud API key — probe for a locally running inference service.
		svcs := r.listRunningServices(ctx)
		var info string
		var err error
		llmClient, info, err = detectLocalLLM(ctx, svcs, r.cfg.Engine.OllamaAddr)
		if err != nil || llmClient == nil {
			slog.Debug("no API key and no local LLM service detected, agent unavailable")
			return nil
		}
		slog.Info("agent using local inference service", "info", info)
	} else {
		switch cfg.LLMProvider {
		case "anthropic":
			llmClient = agentllm.NewAnthropicClient(cfg.LLMModel, cfg.LLMAPIKey)
		case "ollama":
			llmClient = agentllm.NewOllamaClient(cfg.LLMModel, cfg.LLMBaseURL)
		default: // "openai" and OpenAI-compatible endpoints (e.g. Kimi, Azure OpenAI)
			// Strip trailing slash so url construction (baseURL + "/chat/completions") is correct.
			baseURL := strings.TrimRight(cfg.LLMBaseURL, "/")
			llmClient = agentllm.NewOpenAIClient(cfg.LLMModel, cfg.LLMAPIKey, baseURL, cfg.LLMUserAgent)
		}
	}

	mcpAdapter := gateway.NewMCPAdapter(r.gateway)
	toolExecutor := gateway.NewAgentExecutorAdapter(mcpAdapter)
	agentInstance := coreagent.NewAgent(llmClient, toolExecutor, nil, coreagent.AgentOptions{
		MaxTokens: cfg.MaxTokens,
	})

	if err := registry.RegisterAgentDomain(r.registry, agentInstance); err != nil {
		return fmt.Errorf("register agent domain: %w", err)
	}

	slog.Info("agent operator ready",
		"provider", llmClient.Name(),
		"model", llmClient.ModelName(),
	)
	return nil
}

// listRunningServices queries the service store for running services.
// Returns nil if the store is unavailable or returns an error.
func (r *RootCommand) listRunningServices(ctx context.Context) []service.ModelService {
	if r.serviceStore == nil {
		return nil
	}
	svcs, _, err := r.serviceStore.List(ctx, service.ServiceFilter{
		Status: service.ServiceStatusRunning,
	})
	if err != nil {
		return nil
	}
	return svcs
}

func (r *RootCommand) addSubCommands() {
	r.cmd.AddCommand(NewVersionCommand(r))
	r.cmd.AddCommand(NewExecCommand(r))
	r.cmd.AddCommand(NewStartCommand(r))
	r.cmd.AddCommand(NewMCPCommand(r))
	r.cmd.AddCommand(NewModelCommand(r))
	r.cmd.AddCommand(NewInferenceCommand(r))
	r.cmd.AddCommand(NewDeviceCommand(r))
	r.cmd.AddCommand(NewEngineCommand(r))
	r.cmd.AddCommand(NewServiceCommand(r))
	r.cmd.AddCommand(NewWorkflowCommand(r))
	r.cmd.AddCommand(NewCatalogCommand(r))
	r.cmd.AddCommand(NewSkillCommand(r))
	r.cmd.AddCommand(NewAgentCommand(r))
}

func (r *RootCommand) Command() *cobra.Command {
	return r.cmd
}

func (r *RootCommand) Gateway() *gateway.Gateway {
	return r.gateway
}

func (r *RootCommand) Registry() *unit.Registry {
	return r.registry
}

func (r *RootCommand) Config() *config.Config {
	return r.cfg
}

func (r *RootCommand) OutputOptions() *OutputOptions {
	return r.opts
}

func (r *RootCommand) SetOutputWriter(w interface{ Write([]byte) (int, error) }) {
	r.opts.Writer = w
}

func (r *RootCommand) EventBus() eventbus.EventBus {
	return r.eventBus
}

func (r *RootCommand) Execute() error {
	return r.cmd.Execute()
}

func (r *RootCommand) ExecuteContext(ctx context.Context) error {
	return r.cmd.ExecuteContext(ctx)
}

func Execute() {
	root := NewRootCommand()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	if err := root.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

func SetVersion(version, buildDate, gitCommit string) {
	cliVersion = version
	cliBuildDate = buildDate
	cliGitCommit = gitCommit
}

func GetVersion() string {
	return cliVersion
}

func GetBuildDate() string {
	return cliBuildDate
}

func GetGitCommit() string {
	return cliGitCommit
}

