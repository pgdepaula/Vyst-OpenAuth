package company_test

import (
	"testing"

	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
	"github.com/stretchr/testify/assert"
)

func TestCompanyInfo_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		situacao company.CadastralSituation
		expected bool
	}{
		{"Active", company.SituationActive, true},
		{"Suspended", company.SituationSuspended, false},
		{"Inapt", company.SituationInapt, false},
		{"Lowered", company.SituationLowered, false},
		{"Null", company.SituationNull, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &company.CompanyInfo{Situacao: tt.situacao}
			assert.Equal(t, tt.expected, info.IsActive())
		})
	}
}

func TestCompanyInfo_CheckEligibility(t *testing.T) {
	t.Run("Eligible", func(t *testing.T) {
		info := &company.CompanyInfo{Situacao: company.SituationActive}
		assert.NoError(t, info.CheckEligibility())
	})

	t.Run("Ineligible", func(t *testing.T) {
		info := &company.CompanyInfo{Situacao: company.SituationInapt}
		assert.ErrorIs(t, info.CheckEligibility(), company.ErrCompanyInactive)
	})
}

func TestCadastralSituation_IsValid(t *testing.T) {
	assert.True(t, company.SituationActive.IsValid())
	assert.True(t, company.SituationSuspended.IsValid())
	assert.True(t, company.SituationInapt.IsValid())
	assert.True(t, company.SituationLowered.IsValid())
	assert.True(t, company.SituationNull.IsValid())

	assert.False(t, company.CadastralSituation("UNKNOWN").IsValid())
	assert.False(t, company.CadastralSituation("").IsValid())
}
