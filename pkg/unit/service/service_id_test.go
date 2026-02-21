package service

import (
	"testing"
)

func TestParseServiceID(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantSID   ServiceID
		wantError bool
	}{
		{
			name:    "simple model id",
			input:   "svc-vllm-model123",
			wantSID: ServiceID{EngineType: "vllm", ModelID: "model123"},
		},
		{
			name:    "model id with dashes",
			input:   "svc-vllm-my-custom-model",
			wantSID: ServiceID{EngineType: "vllm", ModelID: "my-custom-model"},
		},
		{
			name:    "tts service",
			input:   "svc-tts-466ca5b4",
			wantSID: ServiceID{EngineType: "tts", ModelID: "466ca5b4"},
		},
		{
			name:    "asr service",
			input:   "svc-asr-sensevoice",
			wantSID: ServiceID{EngineType: "asr", ModelID: "sensevoice"},
		},
		{
			name:      "no svc prefix",
			input:     "invalid",
			wantError: true,
		},
		{
			name:      "empty string",
			input:     "",
			wantError: true,
		},
		{
			name:      "only prefix",
			input:     "svc-",
			wantError: true,
		},
		{
			name:      "missing model id",
			input:     "svc-vllm",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseServiceID(tt.input)
			if tt.wantError {
				if err == nil {
					t.Errorf("ParseServiceID(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseServiceID(%q) unexpected error: %v", tt.input, err)
			}
			if got.EngineType != tt.wantSID.EngineType {
				t.Errorf("EngineType: got %q, want %q", got.EngineType, tt.wantSID.EngineType)
			}
			if got.ModelID != tt.wantSID.ModelID {
				t.Errorf("ModelID: got %q, want %q", got.ModelID, tt.wantSID.ModelID)
			}
		})
	}
}

func TestServiceID_String(t *testing.T) {
	sid := ServiceID{EngineType: "vllm", ModelID: "my-custom-model"}
	got := sid.String()
	want := "svc-vllm-my-custom-model"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestParseServiceID_RoundTrip(t *testing.T) {
	cases := []string{
		"svc-vllm-model123",
		"svc-vllm-my-custom-model",
		"svc-tts-466ca5b4",
		"svc-asr-sensevoice",
	}
	for _, original := range cases {
		t.Run(original, func(t *testing.T) {
			sid, err := ParseServiceID(original)
			if err != nil {
				t.Fatalf("ParseServiceID(%q) error: %v", original, err)
			}
			roundTripped := sid.String()
			if roundTripped != original {
				t.Errorf("round-trip: got %q, want %q", roundTripped, original)
			}
			// Parse again to confirm idempotency
			sid2, err := ParseServiceID(roundTripped)
			if err != nil {
				t.Fatalf("ParseServiceID(sid.String()) error: %v", err)
			}
			if sid2 != sid {
				t.Errorf("second parse differs: got %+v, want %+v", sid2, sid)
			}
		})
	}
}
