package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/config"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/provider"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/provider/huggingface"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/store"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/registry"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/engine"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/service"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cliVersion   = "dev"
	cliBuildDate = "unknown"
	cliGitCommit = "unknown"
)

type RootCommand struct {
	cmd       *cobra.Command
	cfg       *config.Config
	gateway   *gateway.Gateway
	registry  *unit.Registry
	opts      *OutputOptions
	formatStr string
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

	viper.BindPFlag("output", pflags.Lookup("output"))
	viper.BindPFlag("quiet", pflags.Lookup("quiet"))
	viper.BindPFlag("config", pflags.Lookup("config"))

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
	serviceProvider := provider.NewHybridServiceProvider(modelStore)
	engineProvider := serviceProvider.GetEngineProvider()

	// Create engine store (memory-based for now)
	engineStore := engine.NewMemoryStore()

	// Register all atomic units with providers
	if err := registry.RegisterAll(r.registry,
		registry.WithModelProvider(modelProvider),
		registry.WithModelStore(modelStore),
		registry.WithServiceProvider(serviceProvider),
		registry.WithServiceStore(serviceStore),
		registry.WithEngineProvider(engineProvider),
		registry.WithEngineStore(engineStore),
	); err != nil {
		return fmt.Errorf("register units: %w", err)
	}

	r.gateway = gateway.NewGateway(r.registry, gateway.WithTimeout(r.cfg.Gateway.RequestTimeoutD))

	return nil
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

func initConfig() (*config.Config, *unit.Registry, *gateway.Gateway, error) {
	cfg, err := config.Load(viper.GetString("config"))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load config: %w", err)
	}

	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry, gateway.WithTimeout(cfg.Gateway.RequestTimeoutD))

	return cfg, registry, gw, nil
}

// func setupEventBus() *eventbus.InMemoryEventBus {
// 	bus := eventbus.NewInMemoryEventBus()
// 	return bus
// }
