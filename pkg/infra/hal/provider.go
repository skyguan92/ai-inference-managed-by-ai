package hal

import "context"

var (
	ErrHardwareNotAvailable = NewProviderError("hardware not available")
	ErrDeviceNotFound       = NewProviderError("device not found")
	ErrCommandFailed        = NewProviderError("command failed")
	ErrPermissionDenied     = NewProviderError("permission denied")
	ErrNotSupported         = NewProviderError("operation not supported")
)

type ProviderError struct {
	Message string
	Cause   error
}

func NewProviderError(message string) *ProviderError {
	return &ProviderError{Message: message}
}

func (e *ProviderError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *ProviderError) Unwrap() error {
	return e.Cause
}

func (e *ProviderError) WithCause(cause error) *ProviderError {
	return &ProviderError{Message: e.Message, Cause: cause}
}

type Provider interface {
	Name() string
	Vendor() string
	Available(ctx context.Context) bool
	Detect(ctx context.Context) ([]HardwareInfo, error)
	GetInfo(ctx context.Context, deviceID string) (*HardwareInfo, error)
	GetMetrics(ctx context.Context, deviceID string) (*HardwareMetrics, error)
	GetHealth(ctx context.Context, deviceID string) (*HardwareHealth, error)
}

type PowerManageable interface {
	SetPowerLimit(ctx context.Context, deviceID string, limitWatts float64) error
	GetPowerLimit(ctx context.Context, deviceID string) (float64, error)
}

type ClockManageable interface {
	SetClockLimit(ctx context.Context, deviceID string, coreClock, memClock uint64) error
	GetClockLimit(ctx context.Context, deviceID string) (coreClock, memClock uint64, err error)
}
