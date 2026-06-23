package company

import (
	"testing"
)

func TestValidateCNPJ(t *testing.T) {
	tests := []struct {
		name     string
		cnpj     string
		expected bool
	}{
		// Valid CNPJs (real format, using known valid test CNPJs)
		{
			name:     "valid CNPJ with formatting",
			cnpj:     "11.222.333/0001-81",
			expected: true,
		},
		{
			name:     "valid CNPJ without formatting",
			cnpj:     "11222333000181",
			expected: true,
		},
		{
			name:     "valid CNPJ Receita Federal example",
			cnpj:     "11444777000161",
			expected: true,
		},

		// Invalid CNPJs
		{
			name:     "invalid CNPJ wrong check digit",
			cnpj:     "11.222.333/0001-82",
			expected: false,
		},
		{
			name:     "invalid CNPJ all zeros",
			cnpj:     "00000000000000",
			expected: false,
		},
		{
			name:     "invalid CNPJ all ones",
			cnpj:     "11111111111111",
			expected: false,
		},
		{
			name:     "invalid CNPJ too short",
			cnpj:     "1122233300018",
			expected: false,
		},
		{
			name:     "invalid CNPJ too long",
			cnpj:     "112223330001811",
			expected: false,
		},
		{
			name:     "invalid CNPJ empty",
			cnpj:     "",
			expected: false,
		},
		{
			name:     "invalid CNPJ with letters",
			cnpj:     "11.222.333/0001-AB",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateCNPJ(tt.cnpj)
			if result != tt.expected {
				t.Errorf("ValidateCNPJ(%q) = %v, expected %v", tt.cnpj, result, tt.expected)
			}
		})
	}
}

func TestNormalizeCNPJ(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already normalized",
			input:    "11222333000181",
			expected: "11222333000181",
		},
		{
			name:     "with dots and slashes",
			input:    "11.222.333/0001-81",
			expected: "11222333000181",
		},
		{
			name:     "with spaces",
			input:    "11 222 333 0001 81",
			expected: "11222333000181",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeCNPJ(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeCNPJ(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatCNPJ(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "unformatted CNPJ",
			input:    "11222333000181",
			expected: "11.222.333/0001-81",
		},
		{
			name:     "already formatted",
			input:    "11.222.333/0001-81",
			expected: "11.222.333/0001-81",
		},
		{
			name:     "invalid length returns input",
			input:    "123",
			expected: "123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCNPJ(tt.input)
			if result != tt.expected {
				t.Errorf("FormatCNPJ(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMaskCNPJ(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid CNPJ",
			input:    "11222333000181",
			expected: "11.222.333/****-**",
		},
		{
			name:     "formatted CNPJ",
			input:    "11.222.333/0001-81",
			expected: "11.222.333/****-**",
		},
		{
			name:     "invalid CNPJ",
			input:    "123",
			expected: "**.***.****/****-**",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskCNPJ(tt.input)
			if result != tt.expected {
				t.Errorf("MaskCNPJ(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsBlacklistedCNPJ(t *testing.T) {
	tests := []struct {
		name     string
		cnpj     string
		expected bool
	}{
		{
			name:     "all zeros is blacklisted",
			cnpj:     "00000000000000",
			expected: true,
		},
		{
			name:     "all nines is blacklisted",
			cnpj:     "99999999999999",
			expected: true,
		},
		{
			name:     "valid CNPJ not blacklisted",
			cnpj:     "11222333000181",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBlacklistedCNPJ(tt.cnpj)
			if result != tt.expected {
				t.Errorf("IsBlacklistedCNPJ(%q) = %v, expected %v", tt.cnpj, result, tt.expected)
			}
		})
	}
}

func TestCompanyRole_IsValid(t *testing.T) {
	tests := []struct {
		role     CompanyRole
		expected bool
	}{
		{RoleAdmin, true},
		{RoleMember, true},
		{RoleViewer, true},
		{CompanyRole("invalid"), false},
		{CompanyRole(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			if tt.role.IsValid() != tt.expected {
				t.Errorf("CompanyRole(%q).IsValid() = %v, expected %v", tt.role, tt.role.IsValid(), tt.expected)
			}
		})
	}
}

func TestNewCompany(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		tenantID    string
		cnpj        string
		razaoSocial string
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid company",
			id:          "test-id",
			tenantID:    "tenant-id",
			cnpj:        "11222333000181",
			razaoSocial: "Test Company",
			wantErr:     false,
		},
		{
			name:        "missing id",
			id:          "",
			tenantID:    "tenant-id",
			cnpj:        "11222333000181",
			razaoSocial: "Test Company",
			wantErr:     true,
			errContains: "company id",
		},
		{
			name:        "missing tenant id",
			id:          "test-id",
			tenantID:    "",
			cnpj:        "11222333000181",
			razaoSocial: "Test Company",
			wantErr:     true,
			errContains: "tenant id",
		},
		{
			name:        "invalid CNPJ",
			id:          "test-id",
			tenantID:    "tenant-id",
			cnpj:        "12345678901234",
			razaoSocial: "Test Company",
			wantErr:     true,
		},
		{
			name:        "missing razao social",
			id:          "test-id",
			tenantID:    "tenant-id",
			cnpj:        "11222333000181",
			razaoSocial: "",
			wantErr:     true,
			errContains: "razão social",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			company, err := NewCompany(tt.id, tt.tenantID, tt.cnpj, tt.razaoSocial)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewCompany() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("NewCompany() unexpected error: %v", err)
				}
				if company == nil {
					t.Error("NewCompany() returned nil company")
					return
				}
				if company.Status != StatusActive {
					t.Errorf("NewCompany() status = %v, expected %v", company.Status, StatusActive)
				}
			}
		})
	}
}

func TestNewCompanyUser(t *testing.T) {
	tests := []struct {
		name      string
		companyID string
		userID    string
		role      CompanyRole
		invitedBy string
		wantErr   bool
	}{
		{
			name:      "valid company user",
			companyID: "company-id",
			userID:    "user-id",
			role:      RoleAdmin,
			invitedBy: "admin-id",
			wantErr:   false,
		},
		{
			name:      "missing company id",
			companyID: "",
			userID:    "user-id",
			role:      RoleAdmin,
			invitedBy: "admin-id",
			wantErr:   true,
		},
		{
			name:      "missing user id",
			companyID: "company-id",
			userID:    "",
			role:      RoleAdmin,
			invitedBy: "admin-id",
			wantErr:   true,
		},
		{
			name:      "invalid role",
			companyID: "company-id",
			userID:    "user-id",
			role:      CompanyRole("invalid"),
			invitedBy: "admin-id",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cu, err := NewCompanyUser(tt.companyID, tt.userID, tt.role, tt.invitedBy)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewCompanyUser() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("NewCompanyUser() unexpected error: %v", err)
				}
				if cu == nil {
					t.Error("NewCompanyUser() returned nil")
					return
				}
				if cu.Status != MembershipActive {
					t.Errorf("NewCompanyUser() status = %v, expected %v", cu.Status, MembershipActive)
				}
			}
		})
	}
}
