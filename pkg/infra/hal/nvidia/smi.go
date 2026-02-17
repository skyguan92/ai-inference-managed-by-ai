package nvidia

import (
	"context"
	"encoding/xml"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/hal"
)

const (
	defaultSMIPath = "nvidia-smi"
	queryFlag      = "--query-gpu="
	formatFlag     = "--format=xml"
)

type SMI struct {
	path    string
	timeout time.Duration
}

func NewSMI(path string) *SMI {
	if path == "" {
		path = defaultSMIPath
	}
	return &SMI{
		path:    path,
		timeout: 10 * time.Second,
	}
}

func (s *SMI) SetTimeout(d time.Duration) {
	s.timeout = d
}

type smiOutput struct {
	AttachedGPUs int `xml:"attached_gpus"`
	GPUs         []struct {
		ID           string `xml:"id,attr"`
		ProductName  string `xml:"product_name"`
		ProductBrand string `xml:"product_brand"`
		UUID         string `xml:"uuid"`
		FanSpeed     string `xml:"fan_speed"`
		Performance  string `xml:"performance_state"`
		Utilization  struct {
			GPUUtil    string `xml:"gpu_util"`
			MemoryUtil string `xml:"memory_util"`
			Encoder    string `xml:"encoder_util"`
			Decoder    string `xml:"decoder_util"`
		} `xml:"utilization"`
		Temperature struct {
			GPUTemp    string `xml:"gpu_temp"`
			GPUTempMax string `xml:"gpu_temp_max_threshold"`
		} `xml:"temperature"`
		PowerReadings struct {
			PowerDraw         string `xml:"power_draw"`
			PowerLimit        string `xml:"power_limit"`
			CurrentPowerLimit string `xml:"current_power_limit"`
		} `xml:"power_readings"`
		FBMemoryUsage struct {
			Total string `xml:"total"`
			Used  string `xml:"used"`
			Free  string `xml:"free"`
		} `xml:"fb_memory_usage"`
		Clocks struct {
			GraphicsClock string `xml:"graphics_clock"`
			MemoryClock   string `xml:"mem_clock"`
		} `xml:"clocks"`
		MaxClocks struct {
			GraphicsClock string `xml:"graphics_clock"`
			MemoryClock   string `xml:"mem_clock"`
		} `xml:"max_clocks"`
		ComputeProcesses struct {
			ProcessInfo []struct {
				PID         string `xml:"pid"`
				ProcessName string `xml:"process_name"`
				UsedMemory  string `xml:"used_memory"`
			} `xml:"process_info"`
		} `xml:"compute_processes"`
	} `xml:"gpu"`
}

func (s *SMI) Available(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.path, "--version")
	return cmd.Run() == nil
}

func (s *SMI) Query(ctx context.Context) (*smiOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.path, "-q", formatFlag)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, hal.ErrCommandFailed.WithCause(
				fmt.Errorf("nvidia-smi exited with code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr)))
		}
		return nil, hal.ErrCommandFailed.WithCause(err)
	}

	var result smiOutput
	if err := xml.Unmarshal(output, &result); err != nil {
		return nil, hal.ErrCommandFailed.WithCause(
			fmt.Errorf("parse nvidia-smi output: %w", err))
	}

	return &result, nil
}

func (s *SMI) QueryDevice(ctx context.Context, deviceID string) (*smiOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.path, "-i", deviceID, "-q", formatFlag)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if strings.Contains(string(exitErr.Stderr), "not found") {
				return nil, hal.ErrDeviceNotFound
			}
			return nil, hal.ErrCommandFailed.WithCause(
				fmt.Errorf("nvidia-smi exited with code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr)))
		}
		return nil, hal.ErrCommandFailed.WithCause(err)
	}

	var result smiOutput
	if err := xml.Unmarshal(output, &result); err != nil {
		return nil, hal.ErrCommandFailed.WithCause(
			fmt.Errorf("parse nvidia-smi output: %w", err))
	}

	return &result, nil
}

func (s *SMI) SetPowerLimit(ctx context.Context, deviceID string, limitWatts uint) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.path, "-i", deviceID, "-pl", fmt.Sprintf("%d", limitWatts))
	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "permission") || strings.Contains(stderr, "Permission") {
				return hal.ErrPermissionDenied.WithCause(err)
			}
			if strings.Contains(stderr, "not supported") || strings.Contains(stderr, "not found") {
				return hal.ErrNotSupported.WithCause(err)
			}
			return hal.ErrCommandFailed.WithCause(
				fmt.Errorf("nvidia-smi -pl failed: %s", string(output)))
		}
		return hal.ErrCommandFailed.WithCause(err)
	}

	return nil
}

func (s *SMI) ResetDevice(ctx context.Context, deviceID string) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.path, "-i", deviceID, "-r")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return hal.ErrCommandFailed.WithCause(
				fmt.Errorf("nvidia-smi -r failed: %s", string(exitErr.Stderr)))
		}
		return hal.ErrCommandFailed.WithCause(err)
	}

	_ = output
	return nil
}

func parsePercentage(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")
	s = strings.TrimSpace(s)
	var v float64
	fmt.Sscanf(s, "%f", &v)
	return v
}

func parseTemperature(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "C")
	s = strings.TrimSpace(s)
	var v float64
	fmt.Sscanf(s, "%f", &v)
	return v
}

func parsePower(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "W")
	s = strings.TrimSpace(s)
	var v float64
	fmt.Sscanf(s, "%f", &v)
	return v
}

func parseMemory(s string) uint64 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "MiB")
	s = strings.TrimSpace(s)
	var v uint64
	fmt.Sscanf(s, "%d", &v)
	return v * 1024 * 1024
}

func parseClock(s string) uint64 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "MHz")
	s = strings.TrimSpace(s)
	var v uint64
	fmt.Sscanf(s, "%d", &v)
	return v
}
