package thinq

import "testing"

func TestPowerOn(t *testing.T) {
	tests := []struct {
		name      string
		state     map[string]interface{}
		wantOn    bool
		wantKnown bool
	}{
		{
			name:      "power on",
			state:     map[string]interface{}{"operation": map[string]interface{}{"airConOperationMode": "POWER_ON"}},
			wantOn:    true,
			wantKnown: true,
		},
		{
			name:      "power off",
			state:     map[string]interface{}{"operation": map[string]interface{}{"airConOperationMode": "POWER_OFF"}},
			wantOn:    false,
			wantKnown: true,
		},
		{
			name:      "power on lowercase",
			state:     map[string]interface{}{"operation": map[string]interface{}{"airConOperationMode": "power_on"}},
			wantOn:    true,
			wantKnown: true,
		},
		{
			name:      "missing operation",
			state:     map[string]interface{}{"temperature": map[string]interface{}{"targetTemperature": 18.0}},
			wantOn:    false,
			wantKnown: false,
		},
		{
			name:      "missing operation mode",
			state:     map[string]interface{}{"operation": map[string]interface{}{}},
			wantOn:    false,
			wantKnown: false,
		},
		{
			name:      "unknown operation mode",
			state:     map[string]interface{}{"operation": map[string]interface{}{"airConOperationMode": "POWER_SAVING"}},
			wantOn:    false,
			wantKnown: false,
		},
		{
			name:      "nil state",
			state:     nil,
			wantOn:    false,
			wantKnown: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			on, known := PowerOn(tc.state)
			if on != tc.wantOn || known != tc.wantKnown {
				t.Errorf("PowerOn() = (%v, %v), want (%v, %v)", on, known, tc.wantOn, tc.wantKnown)
			}
		})
	}
}

func TestTargetTemperature(t *testing.T) {
	tests := []struct {
		name   string
		state  map[string]interface{}
		want   float64
		wantOK bool
	}{
		{
			name:   "target temperature",
			state:  map[string]interface{}{"temperature": map[string]interface{}{"targetTemperature": 18.0}},
			want:   18,
			wantOK: true,
		},
		{
			name:   "integer target temperature",
			state:  map[string]interface{}{"temperature": map[string]interface{}{"targetTemperature": 21}},
			want:   21,
			wantOK: true,
		},
		{
			name:   "missing temperature block",
			state:  map[string]interface{}{"operation": map[string]interface{}{"airConOperationMode": "POWER_ON"}},
			want:   0,
			wantOK: false,
		},
		{
			name:   "current temperature only",
			state:  map[string]interface{}{"temperature": map[string]interface{}{"currentTemperature": 19.0, "unit": "C"}},
			want:   0,
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := TargetTemperature(tc.state)
			if ok != tc.wantOK {
				t.Fatalf("TargetTemperature() ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && got != tc.want {
				t.Errorf("TargetTemperature() = %v, want %v", got, tc.want)
			}
		})
	}
}
