package brasilapi

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
	"github.com/stretchr/testify/assert"
)

// RoundTripFunc .
type RoundTripFunc func(req *http.Request) *http.Response

// RoundTrip .
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// NewTestClient returns *http.Client with Transport replaced to avoid making real calls
func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func TestBrasilAPIAdapter_GetByCNPJ_Success(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(t, "https://mock.brasilapi.com.br/api/cnpj/v1/00000000000191", req.URL.String())
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(bytes.NewBufferString(`{
				"cnpj": "00000000000191",
				"razao_social": "MOCK COMPANY LTDA",
				"descricao_situacao_cadastral": "ATIVA",
				"nome_fantasia": "MOCK FANTASIA",
				"cnae_fiscal": 1234567,
				"natureza_juridica": "Sociedade Empresária Limitada",
				"logradouro": "RUA FAKE",
				"numero": "123",
				"bairro": "CENTRO",
				"municipio": "SÃO PAULO",
				"uf": "SP",
				"cep": "01000000"
			}`)),
			Header: make(http.Header),
		}
	})

	adapter := NewBrasilAPIAdapter(client, "https://mock.brasilapi.com.br/api/cnpj/v1")
	info, err := adapter.GetByCNPJ(context.Background(), "00000000000191")

	assert.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, "00000000000191", info.CNPJ)
	assert.Equal(t, "MOCK COMPANY LTDA", info.RazaoSocial)
}

func TestBrasilAPIAdapter_GetByCNPJ_NotFound(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(bytes.NewBufferString(`{"message": "CNPJ não encontrado."}`)),
			Header:     make(http.Header),
		}
	})

	adapter := NewBrasilAPIAdapter(client, "")
	info, err := adapter.GetByCNPJ(context.Background(), "99999999999999")

	assert.ErrorIs(t, err, company.ErrCompanyInfoNotFound)
	assert.Nil(t, info)
}

func TestBrasilAPIAdapter_GetByCNPJ_HTTPError(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 500,
			Body:       io.NopCloser(bytes.NewBufferString(`Internal Server Error`)),
			Header:     make(http.Header),
		}
	})

	adapter := NewBrasilAPIAdapter(client, "")
	info, err := adapter.GetByCNPJ(context.Background(), "00000000000191")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "brasilapi: unexpected status code 500")
	assert.Nil(t, info)
}
