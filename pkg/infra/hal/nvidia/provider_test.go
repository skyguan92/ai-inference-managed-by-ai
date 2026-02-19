package nvidia

import (
	"context"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/hal"
)

func TestProvider_Name(t *testing.T) {
	p := NewProvider()
	if p.Name() != "nvidia" {
		t.Errorf("expected name 'nvidia', got %s", p.Name())
	}
}

func TestProvider_Vendor(t *testing.T) {
	p := NewProvider()
	if p.Vendor() != hal.VendorNVIDIA {
		t.Errorf("expected vendor '%s', got %s", hal.VendorNVIDIA, p.Vendor())
	}
}

func TestProvider_Available(t *testing.T) {
	p := NewProvider()
	ctx := context.Background()

	result := p.Available(ctx)

	t.Logf("NVIDIA available: %v (this depends on whether nvidia-smi is installed)", result)
}

func TestProvider_Options(t *testing.T) {
	t.Run("WithSMIPath", func(t *testing.T) {
		p := NewProvider(WithSMIPath("/custom/path/nvidia-smi"))
		if smi, ok := p.smi.(*SMI); ok {
			if smi.path != "/custom/path/nvidia-smi" {
				t.Errorf("expected SMI path '/custom/path/nvidia-smi', got %s", smi.path)
			}
		}
	})

	t.Run("WithCacheTTL", func(t *testing.T) {
		p := NewProvider(WithCacheTTL(60 * time.Second))
		if p.cacheTTL != 60*time.Second {
			t.Errorf("expected cache TTL 60s, got %v", p.cacheTTL)
		}
	})

	t.Run("default values", func(t *testing.T) {
		p := NewProvider()
		if smi, ok := p.smi.(*SMI); ok {
			if smi.path != "nvidia-smi" {
				t.Errorf("expected default SMI path 'nvidia-smi', got %s", smi.path)
			}
		}
		if p.cacheTTL != 30*time.Second {
			t.Errorf("expected default cache TTL 30s, got %v", p.cacheTTL)
		}
	})
}

func TestProvider_ExtractIndex(t *testing.T) {
	p := NewProvider()

	tests := []struct {
		deviceID string
		expected int
	}{
		{"nvidia-0", 0},
		{"nvidia-3", 3},
		{"gpu-1", 1},
		{"5", 5},
		{"invalid", -1},
		{"", -1},
	}

	for _, tt := range tests {
		result := p.extractIndex(tt.deviceID)
		if result != tt.expected {
			t.Errorf("extractIndex(%s) = %d, expected %d", tt.deviceID, result, tt.expected)
		}
	}
}

func TestProvider_ToSMIID(t *testing.T) {
	p := NewProvider()

	tests := []struct {
		deviceID string
		expected string
	}{
		{"nvidia-0", "0"},
		{"nvidia-5", "5"},
		{"gpu-2", "2"},
		{"3", "3"},
		{"custom-id", "custom-id"},
	}

	for _, tt := range tests {
		result := p.toSMIID(tt.deviceID)
		if result != tt.expected {
			t.Errorf("toSMIID(%s) = %s, expected %s", tt.deviceID, result, tt.expected)
		}
	}
}

func TestProvider_GetCachedInfo(t *testing.T) {
	p := NewProvider()

	t.Run("no cache entry", func(t *testing.T) {
		p.cache = make(map[string]*hal.HardwareInfo)
		p.cacheTime = time.Now()
		info := p.getCachedInfo("nvidia-0")
		if info != nil {
			t.Error("expected nil for non-existent cache entry")
		}
	})

	t.Run("valid cache entry", func(t *testing.T) {
		p.cache = make(map[string]*hal.HardwareInfo)
		p.cacheTime = time.Now()
		p.cacheTTL = 30 * time.Second

		cachedInfo := &hal.HardwareInfo{
			ID:   "nvidia-0",
			Name: "Test GPU",
		}
		p.cache["nvidia-0"] = cachedInfo

		info := p.getCachedInfo("nvidia-0")
		if info == nil {
			t.Fatal("expected cached info, got nil")
		}
		if info.Name != "Test GPU" {
			t.Errorf("expected name 'Test GPU', got %s", info.Name)
		}
	})

	t.Run("expired cache entry", func(t *testing.T) {
		p.cache = make(map[string]*hal.HardwareInfo)
		p.cacheTime = time.Now().Add(-60 * time.Second)
		p.cacheTTL = 30 * time.Second

		cachedInfo := &hal.HardwareInfo{
			ID:   "nvidia-0",
			Name: "Test GPU",
		}
		p.cache["nvidia-0"] = cachedInfo

		info := p.getCachedInfo("nvidia-0")
		if info != nil {
			t.Error("expected nil for expired cache entry")
		}
	})
}

func TestProvider_SetPowerLimit(t *testing.T) {
	p := NewProvider()

	t.Run("zero power limit", func(t *testing.T) {
		err := p.SetPowerLimit(context.Background(), "nvidia-0", 0)
		if err == nil {
			t.Error("expected error for zero power limit")
		}
	})

	t.Run("negative power limit", func(t *testing.T) {
		err := p.SetPowerLimit(context.Background(), "nvidia-0", -10)
		if err == nil {
			t.Error("expected error for negative power limit")
		}
	})
}

func TestProvider_Detect(t *testing.T) {
	p := NewProvider()

	_, err := p.Detect(context.Background())
	if err != nil {
		t.Logf("Detect failed as expected without nvidia-smi: %v", err)
	}
}

func TestProvider_GetDevice(t *testing.T) {
	p := NewProvider()

	_, err := p.GetDevice(context.Background(), "nvidia-0")
	if err != nil {
		t.Logf("GetDevice failed as expected without nvidia-smi: %v", err)
	}
}

func TestProvider_GetDeviceFromCache(t *testing.T) {
	p := NewProvider()
	p.cache = make(map[string]*hal.HardwareInfo)
	p.cacheTTL = 30 * time.Second
	p.cacheTime = time.Now()

	cachedInfo := &hal.HardwareInfo{
		ID:           "nvidia-0",
		Name:         "NVIDIA GeForce RTX 4090",
		Vendor:       hal.VendorNVIDIA,
		Type:         hal.DeviceTypeGPU,
		Architecture: "Ada Lovelace",
		Memory:       24564 * 1024 * 1024,
	}
	p.cache["nvidia-0"] = cachedInfo

	device, err := p.GetDevice(context.Background(), "nvidia-0")
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}

	if device.ID != "nvidia-0" {
		t.Errorf("expected ID 'nvidia-0', got %s", device.ID)
	}
	if device.Name != "NVIDIA GeForce RTX 4090" {
		t.Errorf("expected name 'NVIDIA GeForce RTX 4090', got %s", device.Name)
	}
	if device.Vendor != hal.VendorNVIDIA {
		t.Errorf("expected vendor '%s', got %s", hal.VendorNVIDIA, device.Vendor)
	}
}

func TestProvider_GetMetrics(t *testing.T) {
	p := NewProvider()

	_, err := p.GetMetrics(context.Background(), "nvidia-0")
	if err != nil {
		t.Logf("GetMetrics failed as expected without nvidia-smi: %v", err)
	}
}

func TestProvider_GetHealth(t *testing.T) {
	p := NewProvider()

	_, err := p.GetHealth(context.Background(), "nvidia-0")
	if err != nil {
		t.Logf("GetHealth failed as expected without nvidia-smi: %v", err)
	}
}

func TestParsePercentage(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"75 %", 75},
		{"50%", 50},
		{" 100 % ", 100},
		{"0%", 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		result := parsePercentage(tt.input)
		if result != tt.expected {
			t.Errorf("parsePercentage(%s) = %f, expected %f", tt.input, result, tt.expected)
		}
	}
}

func TestParseTemperature(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"65 C", 65},
		{"80C", 80},
		{" 45 C ", 45},
		{"0 C", 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		result := parseTemperature(tt.input)
		if result != tt.expected {
			t.Errorf("parseTemperature(%s) = %f, expected %f", tt.input, result, tt.expected)
		}
	}
}

func TestParsePower(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"250.00 W", 250},
		{"450W", 450},
		{" 100.5 W ", 100.5},
		{"0 W", 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		result := parsePower(tt.input)
		if result != tt.expected {
			t.Errorf("parsePower(%s) = %f, expected %f", tt.input, result, tt.expected)
		}
	}
}

func TestParseMemory(t *testing.T) {
	tests := []struct {
		input    string
		expected uint64
	}{
		{"24564 MiB", 24564 * 1024 * 1024},
		{"1024 MiB", 1024 * 1024 * 1024},
		{"0 MiB", 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		result := parseMemory(tt.input)
		if result != tt.expected {
			t.Errorf("parseMemory(%s) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}

func TestParseClock(t *testing.T) {
	tests := []struct {
		input    string
		expected uint64
	}{
		{"2520 MHz", 2520},
		{"1000 MHz", 1000},
		{"0 MHz", 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		result := parseClock(tt.input)
		if result != tt.expected {
			t.Errorf("parseClock(%s) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}

func TestSMI_New(t *testing.T) {
	t.Run("default path", func(t *testing.T) {
		smi := NewSMI("")
		if smi.path != "nvidia-smi" {
			t.Errorf("expected default path 'nvidia-smi', got %s", smi.path)
		}
	})

	t.Run("custom path", func(t *testing.T) {
		smi := NewSMI("/custom/path/nvidia-smi")
		if smi.path != "/custom/path/nvidia-smi" {
			t.Errorf("expected custom path, got %s", smi.path)
		}
	})
}

func TestSMI_SetTimeout(t *testing.T) {
	smi := NewSMI("")
	smi.SetTimeout(30 * time.Second)
	if smi.timeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", smi.timeout)
	}
}

func TestSMI_Available(t *testing.T) {
	smi := NewSMI("")
	result := smi.Available(context.Background())
	t.Logf("SMI available: %v (depends on nvidia-smi installation)", result)
}

func TestSMI_Query_XMLParsing(t *testing.T) {
	smi := NewSMI("")
	smi.SetTimeout(5 * time.Second)

	_, err := smi.Query(context.Background())
	if err != nil {
		t.Logf("Query failed as expected without nvidia-smi: %v", err)
	}
}

func TestSMI_QueryDevice_XMLParsing(t *testing.T) {
	smi := NewSMI("")
	smi.SetTimeout(5 * time.Second)

	_, err := smi.QueryDevice(context.Background(), "0")
	if err != nil {
		t.Logf("QueryDevice failed as expected without nvidia-smi: %v", err)
	}
}

func TestSMI_ResetDevice(t *testing.T) {
	smi := NewSMI("")
	smi.SetTimeout(5 * time.Second)

	err := smi.ResetDevice(context.Background(), "0")
	if err != nil {
		t.Logf("ResetDevice failed as expected without nvidia-smi: %v", err)
	}
}

func TestSMI_SetPowerLimit(t *testing.T) {
	smi := NewSMI("")
	smi.SetTimeout(5 * time.Second)

	err := smi.SetPowerLimit(context.Background(), "0", 250)
	if err != nil {
		t.Logf("SetPowerLimit failed as expected without nvidia-smi: %v", err)
	}
}

type mockSMI struct {
	available     bool
	queryErr      error
	queryResult   *smiOutput
	queryDeviceFn func(ctx context.Context, deviceID string) (*smiOutput, error)
	setPowerErr   error
}

func (m *mockSMI) Available(ctx context.Context) bool {
	return m.available
}

func (m *mockSMI) Query(ctx context.Context) (*smiOutput, error) {
	if m.queryErr != nil {
		return nil, m.queryErr
	}
	return m.queryResult, nil
}

func (m *mockSMI) QueryDevice(ctx context.Context, deviceID string) (*smiOutput, error) {
	if m.queryDeviceFn != nil {
		return m.queryDeviceFn(ctx, deviceID)
	}
	if m.queryErr != nil {
		return nil, m.queryErr
	}
	return m.queryResult, nil
}

func (m *mockSMI) SetPowerLimit(ctx context.Context, deviceID string, limitWatts uint) error {
	return m.setPowerErr
}

func (m *mockSMI) ResetDevice(ctx context.Context, deviceID string) error {
	return nil
}

func TestProvider_Detect_Success(t *testing.T) {
	mock := &mockSMI{
		available:   true,
		queryResult: createSMIOutput("P0", "45 C", "1024 MiB", "24564 MiB"),
	}
	mock.queryResult.AttachedGPUs = 2
	mock.queryResult.GPUs[0].ProductName = "NVIDIA GeForce RTX 4090"
	mock.queryResult.GPUs[0].ProductBrand = "Ada Lovelace"
	mock.queryResult.GPUs = append(mock.queryResult.GPUs, createSMIOutput("P0", "45 C", "512 MiB", "16384 MiB").GPUs[0])
	mock.queryResult.GPUs[1].ID = "1"
	mock.queryResult.GPUs[1].ProductName = "NVIDIA GeForce RTX 4080"
	mock.queryResult.GPUs[1].FBMemoryUsage.Total = "16384 MiB"
	mock.queryResult.GPUs[1].FBMemoryUsage.Used = "512 MiB"
	mock.queryResult.GPUs[1].FBMemoryUsage.Free = "15872 MiB"

	p := NewProvider(withSMI(mock))

	devices, err := p.Detect(context.Background())
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if len(devices) != 2 {
		t.Errorf("expected 2 devices, got %d", len(devices))
	}

	if devices[0].Name != "NVIDIA GeForce RTX 4090" {
		t.Errorf("expected name 'NVIDIA GeForce RTX 4090', got %s", devices[0].Name)
	}

	if devices[1].Name != "NVIDIA GeForce RTX 4080" {
		t.Errorf("expected name 'NVIDIA GeForce RTX 4080', got %s", devices[1].Name)
	}
}

func TestProvider_Detect_Error(t *testing.T) {
	mock := &mockSMI{
		available: false,
		queryErr:  hal.ErrCommandFailed.WithCause(context.DeadlineExceeded),
	}
	p := NewProvider(withSMI(mock))

	_, err := p.Detect(context.Background())
	if err == nil {
		t.Error("expected error when SMI query fails")
	}
}

func TestProvider_GetDevice_WithoutCache(t *testing.T) {
	mock := &mockSMI{
		queryResult: createSMIOutput("P0", "65 C", "2048 MiB", "24576 MiB"),
	}
	mock.queryResult.GPUs[0].ProductName = "NVIDIA GeForce RTX 3090"
	mock.queryResult.GPUs[0].ProductBrand = "Ampere"

	p := NewProvider(withSMI(mock))
	p.cacheTTL = 30 * time.Second
	p.cacheTime = time.Now()

	device, err := p.GetDevice(context.Background(), "nvidia-0")
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}

	if device.Name != "NVIDIA GeForce RTX 3090" {
		t.Errorf("expected name 'NVIDIA GeForce RTX 3090', got %s", device.Name)
	}
}

func TestProvider_GetDevice_NotFound(t *testing.T) {
	mock := &mockSMI{
		queryResult: &smiOutput{
			AttachedGPUs: 0,
			GPUs:         nil,
		},
	}
	p := NewProvider(withSMI(mock))
	p.cacheTTL = 30 * time.Second
	p.cacheTime = time.Now()

	_, err := p.GetDevice(context.Background(), "nvidia-99")
	if err == nil {
		t.Error("expected error when device not found")
	}
}

func TestProvider_GetDevice_QueryError(t *testing.T) {
	mock := &mockSMI{
		queryErr: hal.ErrDeviceNotFound,
	}
	p := NewProvider(withSMI(mock))
	p.cacheTTL = 30 * time.Second
	p.cacheTime = time.Now()

	_, err := p.GetDevice(context.Background(), "nvidia-0")
	if err == nil {
		t.Error("expected error when query fails")
	}
}

func TestProvider_GetMetrics_Success(t *testing.T) {
	mock := &mockSMI{
		queryResult: createSMIOutput("P0", "65 C", "8192 MiB", "24564 MiB"),
	}
	mock.queryResult.GPUs[0].Utilization.GPUUtil = "75 %"
	mock.queryResult.GPUs[0].PowerReadings.PowerDraw = "250.00 W"

	p := NewProvider(withSMI(mock))

	metrics, err := p.GetMetrics(context.Background(), "nvidia-0")
	if err != nil {
		t.Fatalf("GetMetrics failed: %v", err)
	}

	if metrics.Utilization != 75 {
		t.Errorf("expected utilization 75, got %f", metrics.Utilization)
	}
	if metrics.Temperature != 65 {
		t.Errorf("expected temperature 65, got %f", metrics.Temperature)
	}
	if metrics.Power != 250 {
		t.Errorf("expected power 250, got %f", metrics.Power)
	}
}

func TestProvider_GetMetrics_NotFound(t *testing.T) {
	mock := &mockSMI{
		queryResult: &smiOutput{
			AttachedGPUs: 0,
			GPUs:         nil,
		},
	}
	p := NewProvider(withSMI(mock))

	_, err := p.GetMetrics(context.Background(), "nvidia-0")
	if err == nil {
		t.Error("expected error when device not found")
	}
}

func TestProvider_GetMetrics_QueryError(t *testing.T) {
	mock := &mockSMI{
		queryErr: hal.ErrCommandFailed,
	}
	p := NewProvider(withSMI(mock))

	_, err := p.GetMetrics(context.Background(), "nvidia-0")
	if err == nil {
		t.Error("expected error when query fails")
	}
}

func TestProvider_GetHealth_Healthy(t *testing.T) {
	mock := &mockSMI{
		queryResult: createSMIOutput("P0", "45 C", "4096 MiB", "24564 MiB"),
	}
	p := NewProvider(withSMI(mock))

	health, err := p.GetHealth(context.Background(), "nvidia-0")
	if err != nil {
		t.Fatalf("GetHealth failed: %v", err)
	}

	if health.Status != hal.HealthStatusHealthy {
		t.Errorf("expected status %s, got %s", hal.HealthStatusHealthy, health.Status)
	}
}

func TestProvider_GetHealth_LowPerformance(t *testing.T) {
	mock := &mockSMI{
		queryResult: createSMIOutput("P8", "45 C", "4096 MiB", "24564 MiB"),
	}
	p := NewProvider(withSMI(mock))

	health, err := p.GetHealth(context.Background(), "nvidia-0")
	if err != nil {
		t.Fatalf("GetHealth failed: %v", err)
	}

	if health.Status != hal.HealthStatusWarning {
		t.Errorf("expected status %s, got %s", hal.HealthStatusWarning, health.Status)
	}
}

func TestProvider_GetHealth_VeryLowPerformance(t *testing.T) {
	mock := &mockSMI{
		queryResult: createSMIOutput("P12", "45 C", "4096 MiB", "24564 MiB"),
	}
	p := NewProvider(withSMI(mock))

	health, err := p.GetHealth(context.Background(), "nvidia-0")
	if err != nil {
		t.Fatalf("GetHealth failed: %v", err)
	}

	if health.Status != hal.HealthStatusWarning {
		t.Errorf("expected status %s, got %s", hal.HealthStatusWarning, health.Status)
	}
}

func TestProvider_GetHealth_ElevatedTemperature(t *testing.T) {
	mock := &mockSMI{
		queryResult: createSMIOutput("P0", "85 C", "4096 MiB", "24564 MiB"),
	}
	p := NewProvider(withSMI(mock))

	health, err := p.GetHealth(context.Background(), "nvidia-0")
	if err != nil {
		t.Fatalf("GetHealth failed: %v", err)
	}

	if health.Status != hal.HealthStatusWarning {
		t.Errorf("expected status %s, got %s", hal.HealthStatusWarning, health.Status)
	}
}

func TestProvider_GetHealth_HighTemperature(t *testing.T) {
	mock := &mockSMI{
		queryResult: createSMIOutput("P0", "95 C", "4096 MiB", "24564 MiB"),
	}
	p := NewProvider(withSMI(mock))

	health, err := p.GetHealth(context.Background(), "nvidia-0")
	if err != nil {
		t.Fatalf("GetHealth failed: %v", err)
	}

	if health.Status != hal.HealthStatusCritical {
		t.Errorf("expected status %s, got %s", hal.HealthStatusCritical, health.Status)
	}
}

func TestProvider_GetHealth_HighMemory(t *testing.T) {
	mock := &mockSMI{
		queryResult: createSMIOutput("P0", "45 C", "24000 MiB", "24564 MiB"),
	}
	p := NewProvider(withSMI(mock))

	health, err := p.GetHealth(context.Background(), "nvidia-0")
	if err != nil {
		t.Fatalf("GetHealth failed: %v", err)
	}

	if health.Status != hal.HealthStatusWarning {
		t.Errorf("expected status %s, got %s", hal.HealthStatusWarning, health.Status)
	}
}

func TestProvider_GetHealth_MultipleIssues(t *testing.T) {
	mock := &mockSMI{
		queryResult: createSMIOutput("P8", "95 C", "24000 MiB", "24564 MiB"),
	}
	p := NewProvider(withSMI(mock))

	health, err := p.GetHealth(context.Background(), "nvidia-0")
	if err != nil {
		t.Fatalf("GetHealth failed: %v", err)
	}

	if health.Status != hal.HealthStatusCritical {
		t.Errorf("expected status %s, got %s", hal.HealthStatusCritical, health.Status)
	}

	if len(health.Issues) < 2 {
		t.Errorf("expected multiple issues, got %d", len(health.Issues))
	}
}

func TestProvider_GetHealth_NotFound(t *testing.T) {
	mock := &mockSMI{
		queryResult: &smiOutput{
			AttachedGPUs: 0,
			GPUs:         nil,
		},
	}
	p := NewProvider(withSMI(mock))

	_, err := p.GetHealth(context.Background(), "nvidia-0")
	if err == nil {
		t.Error("expected error when device not found")
	}
}

func TestProvider_GetHealth_QueryError(t *testing.T) {
	mock := &mockSMI{
		queryErr: hal.ErrCommandFailed,
	}
	p := NewProvider(withSMI(mock))

	_, err := p.GetHealth(context.Background(), "nvidia-0")
	if err == nil {
		t.Error("expected error when query fails")
	}
}

func TestProvider_SetPowerLimit_Success(t *testing.T) {
	mock := &mockSMI{}
	p := NewProvider(withSMI(mock))

	err := p.SetPowerLimit(context.Background(), "nvidia-0", 250)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestProvider_SetPowerLimit_SMIError(t *testing.T) {
	mock := &mockSMI{
		setPowerErr: hal.ErrPermissionDenied,
	}
	p := NewProvider(withSMI(mock))

	err := p.SetPowerLimit(context.Background(), "nvidia-0", 250)
	if err == nil {
		t.Error("expected error from SMI")
	}
}

func createSMIOutput(performance, temp, memUsed, memTotal string) *smiOutput {
	return &smiOutput{
		AttachedGPUs: 1,
		GPUs: []struct {
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
		}{
			{
				ID:           "0",
				ProductName:  "NVIDIA GeForce RTX 4090",
				ProductBrand: "Ada Lovelace",
				Performance:  performance,
				Temperature: struct {
					GPUTemp    string `xml:"gpu_temp"`
					GPUTempMax string `xml:"gpu_temp_max_threshold"`
				}{GPUTemp: temp},
				FBMemoryUsage: struct {
					Total string `xml:"total"`
					Used  string `xml:"used"`
					Free  string `xml:"free"`
				}{Total: memTotal, Used: memUsed},
			},
		},
	}
}
