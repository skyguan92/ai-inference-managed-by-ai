package cli

import (
	"context"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/spf13/cobra"
)

func NewDeviceCommand(root *RootCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "device",
		Short: "Device management commands",
		Long: `Manage hardware devices in the AIMA infrastructure.

This includes detecting devices, viewing device information,
and monitoring device metrics.`,
	}

	cmd.AddCommand(NewDeviceDetectCommand(root))
	cmd.AddCommand(NewDeviceInfoCommand(root))
	cmd.AddCommand(NewDeviceMetricsCommand(root))

	return cmd
}

func NewDeviceDetectCommand(root *RootCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "detect",
		Short: "Detect hardware devices",
		Long: `Detect all available hardware devices on the system.

This scans for GPUs, NPUs, and other accelerators.`,
		Example: `  # Detect all devices
  aima device detect

  # With JSON output
  aima device detect --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeviceDetect(cmd.Context(), root)
		},
	}

	return cmd
}

func runDeviceDetect(ctx context.Context, root *RootCommand) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	req := &gateway.Request{
		Type:  gateway.TypeCommand,
		Unit:  "device.detect",
		Input: map[string]any{},
	}

	resp := gw.Handle(ctx, req)

	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("device detection failed: %s", resp.Error.Message)
	}

	return PrintOutput(resp.Data, opts)
}

func NewDeviceInfoCommand(root *RootCommand) *cobra.Command {
	var deviceID string

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Get device information",
		Long: `Get detailed information about a specific device.

If no device ID is specified, shows info for all devices.`,
		Example: `  # Get info for all devices
  aima device info

  # Get info for specific device
  aima device info --id gpu0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeviceInfo(cmd.Context(), root, deviceID)
		},
	}

	cmd.Flags().StringVarP(&deviceID, "id", "i", "", "Device ID")

	return cmd
}

func runDeviceInfo(ctx context.Context, root *RootCommand, deviceID string) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	input := map[string]any{}
	if deviceID != "" {
		input["device_id"] = deviceID
	}

	req := &gateway.Request{
		Type:  gateway.TypeQuery,
		Unit:  "device.info",
		Input: input,
	}

	resp := gw.Handle(ctx, req)

	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("get device info failed: %s", resp.Error.Message)
	}

	return PrintOutput(resp.Data, opts)
}

func NewDeviceMetricsCommand(root *RootCommand) *cobra.Command {
	var deviceID string
	var history bool

	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "Get device metrics",
		Long: `Get real-time metrics for a device.

Shows utilization, temperature, power consumption, and memory usage.`,
		Example: `  # Get current metrics
  aima device metrics

  # Get metrics for specific device
  aima device metrics --id gpu0

  # Include historical data
  aima device metrics --history`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeviceMetrics(cmd.Context(), root, deviceID, history)
		},
	}

	cmd.Flags().StringVarP(&deviceID, "id", "i", "", "Device ID")
	cmd.Flags().BoolVar(&history, "history", false, "Include historical metrics")

	return cmd
}

func runDeviceMetrics(ctx context.Context, root *RootCommand, deviceID string, history bool) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	input := map[string]any{}
	if deviceID != "" {
		input["device_id"] = deviceID
	}
	if history {
		input["history"] = true
	}

	req := &gateway.Request{
		Type:  gateway.TypeQuery,
		Unit:  "device.metrics",
		Input: input,
	}

	resp := gw.Handle(ctx, req)

	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("get device metrics failed: %s", resp.Error.Message)
	}

	return PrintOutput(resp.Data, opts)
}
