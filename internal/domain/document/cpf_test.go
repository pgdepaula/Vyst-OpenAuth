package document

import (
	"encoding/json"
	"testing"
)

func TestNewCPF(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr error
	}{
		{"valid formatted", "529.982.247-25", nil},
		{"valid unformatted", "52998224725", nil},
		{"invalid checksum", "529.982.247-26", ErrCPFInvalid},
		{"blacklisted", "111.111.111-11", ErrCPFBlacklisted},
		{"empty", "", ErrCPFInvalid},
		{"too short", "123", ErrCPFInvalid},
		{"too long", "1234567890123", ErrCPFInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCPF(tt.value)
			if err != tt.wantErr {
				t.Errorf("NewCPF(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
			if err == nil {
				if got.IsEmpty() {
					t.Error("expected non-empty CPF")
				}
				if got.String() != "529.982.247-25" && tt.value != "" {
					t.Errorf("expected formatted string to be standard, got %q", got.String())
				}
			}
		})
	}
}

func TestCPF_JSON(t *testing.T) {
	cpf, _ := NewCPF("52998224725")

	// Marshal
	bytes, err := json.Marshal(cpf)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	expected := `"529.982.247-25"`
	if string(bytes) != expected {
		t.Errorf("MarshalJSON = %s, want %s", string(bytes), expected)
	}

	// Unmarshal
	var got CPF
	if err := json.Unmarshal(bytes, &got); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if !got.Equals(cpf) {
		t.Errorf("UnmarshalJSON = %v, want %v", got, cpf)
	}
}

func TestCPF_Value(t *testing.T) {
	cpf, _ := NewCPF("52998224725")
	val, err := cpf.Value()
	if err != nil {
		t.Fatalf("Value() failed: %v", err)
	}
	if val.(string) != "52998224725" {
		t.Errorf("Value() = %v, want %q", val, "52998224725")
	}
}

func TestCPF_Scan(t *testing.T) {
	var cpf CPF
	err := cpf.Scan("52998224725")
	if err != nil {
		t.Fatalf("Scan() failed: %v", err)
	}
	expected, _ := NewCPF("52998224725")
	if !cpf.Equals(expected) {
		t.Errorf("Scan() = %v, want %v", cpf, expected)
	}

	err = cpf.Scan("invalid")
	if err == nil {
		t.Error("Scan() expected error for invalid CPF")
	}
}

func TestLegacyFunctions(t *testing.T) {
	// Identify if legacy functions still behave as expected
	if err := ValidateCPF("52998224725"); err != nil {
		t.Error("ValidateCPF failed for valid CPF")
	}
	if FormatCPF("52998224725") != "529.982.247-25" {
		t.Error("FormatCPF failed")
	}
	if NormalizeCPF("529.982.247-25") != "52998224725" {
		t.Error("NormalizeCPF failed")
	}
}
