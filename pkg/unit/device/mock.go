package device

import "context"

// MockProvider implements DeviceProvider for testing purposes.
type MockProvider struct {
	Devices    []DeviceInfo
	Health     *DeviceHealth
	Metrics    *DeviceMetrics
	Err        error
	PowerLimit float64
	PowerSetID string
}

func NewMockProvider() *MockProvider {
	return &MockProvider{
		Devices: []DeviceInfo{
			{
				ID:           "gpu-0",
				Name:         "Mock GPU",
				Vendor:       "MockVendor",
				Type:         "gpu",
				Architecture: "mock-arch",
				Memory:       24564,
				Capabilities: []string{"cuda", "tensor"},
			},
		},
		Health: &DeviceHealth{
			Status: "healthy",
			Issues: []string{},
		},
		Metrics: &DeviceMetrics{
			Utilization: 50.0,
			Temperature: 60.0,
			Power:       200.0,
			MemoryUsed:  8192000000,
			MemoryTotal: 24564000000,
		},
	}
}

func (m *MockProvider) Detect(ctx context.Context) ([]DeviceInfo, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Devices, nil
}

func (m *MockProvider) GetDevice(ctx context.Context, deviceID string) (*DeviceInfo, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	for _, d := range m.Devices {
		if d.ID == deviceID {
			return &d, nil
		}
	}
	return nil, ErrDeviceNotFound
}

func (m *MockProvider) GetMetrics(ctx context.Context, deviceID string) (*DeviceMetrics, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Metrics, nil
}

func (m *MockProvider) GetHealth(ctx context.Context, deviceID string) (*DeviceHealth, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Health, nil
}

func (m *MockProvider) SetPowerLimit(ctx context.Context, deviceID string, limitWatts float64) error {
	if m.Err != nil {
		return m.Err
	}
	m.PowerSetID = deviceID
	m.PowerLimit = limitWatts
	return nil
}
