package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/docker"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/eventbus"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/engine"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/service"
	"github.com/spf13/cobra"
)

// serviceStopTimeout is the per-request timeout for service stop commands.
// Stopping a container may require a graceful shutdown window (up to 30s before
// SIGKILL) plus docker rm, so we need significantly more than the default 30s.
const serviceStopTimeout = 2 * time.Minute

func NewServiceCommand(root *RootCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Service management commands",
		Long: `Manage AI model inference services.

This includes starting, stopping, and listing model inference services.`,
	}

	cmd.AddCommand(NewServiceStartCommand(root))
	cmd.AddCommand(NewServiceStopCommand(root))
	cmd.AddCommand(NewServiceStatusCommand(root))
	cmd.AddCommand(NewServiceListCommand(root))
	cmd.AddCommand(NewServiceCreateCommand(root))
	cmd.AddCommand(NewServiceLogsCommand(root))
	cmd.AddCommand(NewServiceCleanupCommand(root))

	return cmd
}

func NewServiceCreateCommand(root *RootCommand) *cobra.Command {
	var (
		modelID   string
		device    string
		port      int
		gpuLayers int
	)

	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a model inference service",
		Long: `Create a new inference service for a model.

This creates a service configuration without starting it.`,
		Example: `  # Create service for a model
  aima service create my-service --model model-xxx

  # Create with specific device
  aima service create my-service --model model-xxx --device gpu

  # Create with custom port
  aima service create my-service --model model-xxx --port 8080`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			return runServiceCreate(cmd.Context(), root, name, modelID, device, port, gpuLayers)
		},
	}

	cmd.Flags().StringVarP(&modelID, "model", "m", "", "Model ID (required)")
	cmd.Flags().StringVarP(&device, "device", "d", "gpu", "Device type (cpu, gpu)")
	cmd.Flags().IntVarP(&port, "port", "p", 0, "Service port (auto-assigned if not specified)")
	cmd.Flags().IntVar(&gpuLayers, "gpu-layers", -1, "Number of GPU layers (-1 for auto)")
	_ = cmd.MarkFlagRequired("model")

	return cmd
}

func runServiceCreate(ctx context.Context, root *RootCommand, name, modelID, device string, port, gpuLayers int) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	// Get model info first
	modelReq := &gateway.Request{
		Type: gateway.TypeQuery,
		Unit: "model.get",
		Input: map[string]any{
			"model_id": modelID,
		},
	}

	modelResp := gw.Handle(ctx, modelReq)
	if !modelResp.Success {
		PrintError(fmt.Errorf("model not found: %s", modelID), opts)
		return fmt.Errorf("model not found: %s", modelID)
	}

	// Determine resource class based on device
	resourceClass := service.ResourceClassMedium
	if device == "gpu" {
		resourceClass = service.ResourceClassLarge
	}

	input := map[string]any{
		"model_id":       modelID,
		"resource_class": resourceClass,
		"replicas":       1,
		"persistent":     true,
	}

	if name != "" {
		input["name"] = name
	}
	if port > 0 {
		input["port"] = port
	}
	if gpuLayers >= 0 {
		input["gpu_layers"] = gpuLayers
	}

	req := &gateway.Request{
		Type:  gateway.TypeCommand,
		Unit:  "service.create",
		Input: input,
	}

	resp := gw.Handle(ctx, req)

	if !resp.Success {
		errMsg := fmt.Sprintf("%s: %s", resp.Error.Code, resp.Error.Message)
		if resp.Error.Details != nil {
			errMsg = fmt.Sprintf("%s\ndetails: %v", errMsg, resp.Error.Details)
		}
		PrintError(fmt.Errorf("%s", errMsg), opts)
		return fmt.Errorf("create service failed: %s", resp.Error.Message)
	}

	PrintSuccess(fmt.Sprintf("Service created for model %s", modelID), opts)
	return PrintOutput(resp.Data, opts)
}

func NewServiceStartCommand(root *RootCommand) *cobra.Command {
	var (
		wait    bool
		timeout int
		async   bool
	)

	cmd := &cobra.Command{
		Use:   "start <service-id>",
		Short: "Start a model inference service",
		Long: `Start an inference service for a model.

This will start the Docker container or native process for the model.`,
		Example: `  # Start a service and wait for ready
  aima service start svc-vlm-model-xxx

  # Start in async mode (don't wait for readiness)
  aima service start svc-vlm-model-xxx --async

  # Start with custom timeout
  aima service start svc-vlm-model-xxx --wait --timeout 300`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// If async mode, don't wait
			if async {
				wait = false
			}
			return runServiceStart(cmd.Context(), root, args[0], wait, timeout, async)
		},
	}

	cmd.Flags().BoolVarP(&wait, "wait", "w", true, "Wait for service to be ready")
	cmd.Flags().IntVarP(&timeout, "timeout", "t", 300, "Timeout in seconds")
	cmd.Flags().BoolVarP(&async, "async", "a", false, "Start in async mode (don't wait for readiness)")

	return cmd
}

func runServiceStart(ctx context.Context, root *RootCommand, serviceID string, wait bool, timeout int, async bool) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	// Subscribe to progress events if waiting synchronously
	var subID eventbus.SubscriptionID
	if wait && !async && root.EventBus() != nil {
		subID, _ = root.EventBus().Subscribe(
			func(event unit.Event) error {
				payload, ok := event.Payload().(map[string]any)
				if !ok {
					return nil
				}
				phase, _ := payload["phase"].(string)
				message, _ := payload["message"].(string)
				progress, _ := payload["progress"].(int)

				switch phase {
				case "pulling":
					fmt.Printf("  [pull] %s\n", message)
				case "starting":
					fmt.Printf("  [start] %s\n", message)
				case "loading":
					fmt.Printf("  [load] %s\n", message)
				case "ready":
					fmt.Printf("  [ready] %s\n", message)
				case "failed":
					fmt.Printf("  [FAIL] %s\n", message)
				default:
					if progress >= 0 {
						fmt.Printf("  [%d%%] %s\n", progress, message)
					} else {
						fmt.Printf("  %s\n", message)
					}
				}
				return nil
			},
			eventbus.FilterByType(engine.EventTypeStartProgress),
		)
		if subID != "" {
			defer root.EventBus().Unsubscribe(subID) //nolint:errcheck
		}
	}

	req := &gateway.Request{
		Type: gateway.TypeCommand,
		Unit: "service.start",
		Input: map[string]any{
			"service_id": serviceID,
			"wait":       wait,
			"timeout":    timeout,
			"async":      async,
		},
		Options: gateway.RequestOptions{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}

	resp := gw.Handle(ctx, req)

	if !resp.Success {
		errMsg := fmt.Sprintf("%s: %s", resp.Error.Code, resp.Error.Message)
		if resp.Error.Details != nil {
			errMsg = fmt.Sprintf("%s\ndetails: %v", errMsg, resp.Error.Details)
		}
		PrintError(fmt.Errorf("%s", errMsg), opts)
		return fmt.Errorf("start service failed: %s", resp.Error.Message)
	}

	if async {
		PrintSuccess(fmt.Sprintf("Service %s started in async mode", serviceID), opts)
		fmt.Println("\nUse 'aima service status <service-id>' to check loading progress")
		fmt.Println("Use 'docker logs' to view detailed logs")
	} else {
		PrintSuccess(fmt.Sprintf("Service %s started successfully", serviceID), opts)
	}
	return PrintOutput(resp.Data, opts)
}

func NewServiceStopCommand(root *RootCommand) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "stop <service-id>",
		Short: "Stop a model inference service",
		Long: `Stop a running inference service.

By default, performs a graceful shutdown. Use --force for immediate termination.`,
		Example: `  # Stop a service gracefully
  aima service stop svc-vlm-model-xxx

  # Force stop
  aima service stop svc-vlm-model-xxx --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServiceStop(cmd.Context(), root, args[0], force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force stop the service")

	return cmd
}

func runServiceStop(ctx context.Context, root *RootCommand, serviceID string, force bool) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	req := &gateway.Request{
		Type: gateway.TypeCommand,
		Unit: "service.stop",
		Input: map[string]any{
			"service_id": serviceID,
			"force":      force,
		},
		Options: gateway.RequestOptions{
			Timeout: serviceStopTimeout,
		},
	}

	resp := gw.Handle(ctx, req)

	if !resp.Success {
		errMsg := fmt.Sprintf("%s: %s", resp.Error.Code, resp.Error.Message)
		if resp.Error.Details != nil {
			errMsg = fmt.Sprintf("%s\ndetails: %v", errMsg, resp.Error.Details)
		}
		PrintError(fmt.Errorf("%s", errMsg), opts)
		return fmt.Errorf("stop service failed: %s", resp.Error.Message)
	}

	PrintSuccess(fmt.Sprintf("Service %s stopped successfully", serviceID), opts)
	return nil
}

func NewServiceStatusCommand(root *RootCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <service-id>",
		Short: "Check service status and loading progress",
		Long: `Check the status of a model inference service.

For services started with --async, this shows loading progress and health status.`,
		Example: `  # Check service status
  aima service status svc-vlm-model-xxx

  # Watch status updates
  watch -n 5 "aima service status svc-vlm-model-xxx"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServiceStatus(cmd.Context(), root, args[0])
		},
	}

	return cmd
}

func runServiceStatus(ctx context.Context, root *RootCommand, serviceID string) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	req := &gateway.Request{
		Type: gateway.TypeQuery,
		Unit: "service.status",
		Input: map[string]any{
			"service_id": serviceID,
		},
	}

	resp := gw.Handle(ctx, req)

	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("check status failed: %s", resp.Error.Message)
	}

	return PrintOutput(resp.Data, opts)
}

func NewServiceListCommand(root *RootCommand) *cobra.Command {
	var (
		status string
		model  string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List model inference services",
		Long:    `List all model inference services and their status.`,
		Example: `  # List all services
  aima service list

  # List only running services
  aima service list --status running

  # List services for a specific model
  aima service list --model model-xxx`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServiceList(cmd.Context(), root, status, model)
		},
	}

	cmd.Flags().StringVarP(&status, "status", "s", "", "Filter by status (running, stopped, error)")
	cmd.Flags().StringVarP(&model, "model", "m", "", "Filter by model ID")

	return cmd
}

func runServiceList(ctx context.Context, root *RootCommand, status, model string) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	input := map[string]any{}
	if status != "" {
		input["status"] = status
	}
	if model != "" {
		input["model_id"] = model
	}

	req := &gateway.Request{
		Type:  gateway.TypeQuery,
		Unit:  "service.list",
		Input: input,
	}

	resp := gw.Handle(ctx, req)

	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("list services failed: %s", resp.Error.Message)
	}

	return PrintOutput(resp.Data, opts)
}

func NewServiceLogsCommand(root *RootCommand) *cobra.Command {
	var (
		follow bool
		tail   int
	)

	cmd := &cobra.Command{
		Use:   "logs <service-id>",
		Short: "View logs for a model inference service",
		Long:  `View logs for a running inference service container.`,
		Example: `  # View last 100 lines of logs
  aima service logs svc-vllm-model-xxx

  # Follow logs in real-time
  aima service logs svc-vllm-model-xxx --follow

  # View last 50 lines
  aima service logs svc-vllm-model-xxx --tail 50`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServiceLogs(cmd.Context(), root, args[0], follow, tail)
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	cmd.Flags().IntVarP(&tail, "tail", "n", 100, "Number of lines to show from the end")

	return cmd
}

func runServiceLogs(ctx context.Context, root *RootCommand, serviceID string, follow bool, tail int) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	req := &gateway.Request{
		Type: gateway.TypeQuery,
		Unit: "service.logs",
		Input: map[string]any{
			"service_id": serviceID,
			"follow":     follow,
			"tail":       tail,
		},
	}

	resp := gw.Handle(ctx, req)
	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("get logs failed: %s", resp.Error.Message)
	}

	if data, ok := resp.Data.(map[string]any); ok {
		if logs, ok := data["logs"].(string); ok {
			fmt.Print(logs)
			return nil
		}
	}
	return PrintOutput(resp.Data, opts)
}

// NewServiceCleanupCommand creates the `aima service cleanup` command which stops
// and removes all AIMA-managed Docker containers and resets service status.
func NewServiceCleanupCommand(root *RootCommand) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Stop and remove all AIMA-managed containers",
		Long: `Stop and remove all Docker containers managed by AIMA (aima.managed=true label).
Also stops any running services tracked in the service store.

Use --force to skip confirmation.`,
		Example: `  # Cleanup all AIMA containers (with confirmation)
  aima service cleanup

  # Force cleanup without confirmation
  aima service cleanup --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServiceCleanup(cmd.Context(), root, force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")
	return cmd
}

func runServiceCleanup(ctx context.Context, root *RootCommand, force bool) error {
	opts := root.OutputOptions()
	gw := root.Gateway()

	if !force {
		fmt.Fprintln(opts.Writer, "This will stop and remove all AIMA-managed Docker containers.")
		fmt.Fprint(opts.Writer, "Continue? [y/N]: ")
		var answer string
		fmt.Scanln(&answer) //nolint:errcheck
		if answer != "y" && answer != "Y" {
			fmt.Fprintln(opts.Writer, "Aborted.")
			return nil
		}
	}

	// Bail early with a clear message when Docker is not running.
	if err := docker.CheckDocker(); err != nil {
		fmt.Fprintln(opts.Writer, "Docker is not available — nothing to clean up.")
		return nil
	}

	// Create a Docker client to list and stop AIMA containers directly.
	var dc docker.Client
	if sdkClient, err := docker.NewSDKClient(); err == nil {
		dc = sdkClient
	} else {
		dc = docker.NewSimpleClient()
	}

	containerIDs, err := dc.ListContainers(ctx, map[string]string{})
	if err != nil {
		PrintError(fmt.Errorf("failed to list containers: %w", err), opts)
		return err
	}

	containerCount := 0
	for _, cid := range containerIDs {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if stopErr := dc.StopContainer(cleanupCtx, cid, 10); stopErr != nil {
			fmt.Fprintf(opts.Writer, "Warning: failed to stop container %s: %v\n", cid, stopErr)
		} else {
			containerCount++
		}
		cancel()
	}

	// Also stop running services tracked in the gateway.
	svcResp := gw.Handle(ctx, &gateway.Request{
		Type:  gateway.TypeQuery,
		Unit:  "service.list",
		Input: map[string]any{},
	})
	serviceCount := 0
	if svcResp.Success {
		if data, ok := svcResp.Data.(map[string]any); ok {
			if items, ok := data["items"].([]any); ok {
				for _, item := range items {
					svc, ok := item.(map[string]any)
					if !ok {
						continue
					}
					svcID, _ := svc["id"].(string)
					svcStatus, _ := svc["status"].(string)
					if svcID == "" || svcStatus == "stopped" {
						continue
					}
					stopResp := gw.Handle(ctx, &gateway.Request{
						Type:  gateway.TypeCommand,
						Unit:  "service.stop",
						Input: map[string]any{"service_id": svcID},
						Options: gateway.RequestOptions{
							Timeout: serviceStopTimeout,
						},
					})
					if stopResp.Success {
						serviceCount++
					}
				}
			}
		}
	}

	fmt.Fprintf(opts.Writer, "Cleaned up %d container(s), reset %d service(s).\n", containerCount, serviceCount)
	return nil
}
