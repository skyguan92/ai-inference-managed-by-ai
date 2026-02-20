package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
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
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
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

	req := &gateway.Request{
		Type: gateway.TypeCommand,
		Unit: "service.start",
		Input: map[string]any{
			"service_id": serviceID,
			"wait":       wait,
			"timeout":    timeout,
			"async":      async,
		},
	}

	resp := gw.Handle(ctx, req)

	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
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
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
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
