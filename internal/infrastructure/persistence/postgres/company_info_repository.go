package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
)

// CompanyInfoRepository implements company.CompanyInfoRepository using PostgreSQL.
type CompanyInfoRepository struct {
	pool *pgxpool.Pool
}

// NewCompanyInfoRepository creates a new CompanyInfoRepository.
func NewCompanyInfoRepository(pool *pgxpool.Pool) *CompanyInfoRepository {
	return &CompanyInfoRepository{pool: pool}
}

// Save persists company info to the cache storage.
func (r *CompanyInfoRepository) Save(ctx context.Context, info *company.CompanyInfo) error {
	query := `
		INSERT INTO company_info_cache (
			cnpj, razao_social, nome_fantasia, situacao, natureza_juridica,
			data_abertura, logradouro, numero, complemento, bairro, cidade, uf, cep,
			telefones, emails, cnae_principal, last_fetched_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)
		ON CONFLICT (cnpj) DO UPDATE SET
			razao_social = EXCLUDED.razao_social,
			nome_fantasia = EXCLUDED.nome_fantasia,
			situacao = EXCLUDED.situacao,
			natureza_juridica = EXCLUDED.natureza_juridica,
			data_abertura = EXCLUDED.data_abertura,
			logradouro = EXCLUDED.logradouro,
			numero = EXCLUDED.numero,
			complemento = EXCLUDED.complemento,
			bairro = EXCLUDED.bairro,
			cidade = EXCLUDED.cidade,
			uf = EXCLUDED.uf,
			cep = EXCLUDED.cep,
			telefones = EXCLUDED.telefones,
			emails = EXCLUDED.emails,
			cnae_principal = EXCLUDED.cnae_principal,
			last_fetched_at = EXCLUDED.last_fetched_at
	`

	telefonesData, _ := json.Marshal(info.Telefones)
	emailsData, _ := json.Marshal(info.Emails)

	_, err := GetExecutor(ctx, r.pool).Exec(ctx, query,
		info.CNPJ,
		info.RazaoSocial,
		nullableString(info.NomeFantasia),
		string(info.Situacao),
		nullableString(info.NaturezaJuridica),
		info.DataAbertura,
		nullableString(info.Endereco.Logradouro),
		nullableString(info.Endereco.Numero),
		nullableString(info.Endereco.Complemento),
		nullableString(info.Endereco.Bairro),
		nullableString(info.Endereco.Cidade),
		nullableString(info.Endereco.UF),
		nullableString(info.Endereco.CEP),
		telefonesData,
		emailsData,
		nullableString(info.CNAEPrincipal),
		info.LastFetchedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save company info cache: %w", err)
	}

	return nil
}

// GetByCNPJ retrieves company info by CNPJ from the cache.
func (r *CompanyInfoRepository) GetByCNPJ(ctx context.Context, cnpj string) (*company.CompanyInfo, error) {
	query := `
		SELECT cnpj, razao_social, nome_fantasia, situacao, natureza_juridica,
		       data_abertura, logradouro, numero, complemento, bairro, cidade, uf, cep,
		       telefones, emails, cnae_principal, last_fetched_at
		FROM company_info_cache
		WHERE cnpj = $1
	`
	row := GetExecutor(ctx, r.pool).QueryRow(ctx, query, cnpj)

	info, err := r.scanCompanyInfo(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, company.ErrCompanyInfoNotFound
		}
		return nil, err
	}

	return info, nil
}

// SearchByName performs a text search on RazaoSocial or NomeFantasia.
func (r *CompanyInfoRepository) SearchByName(ctx context.Context, query string, limit int) ([]*company.CompanyInfo, error) {
	sqlQuery := `
		SELECT cnpj, razao_social, nome_fantasia, situacao, natureza_juridica,
		       data_abertura, logradouro, numero, complemento, bairro, cidade, uf, cep,
		       telefones, emails, cnae_principal, last_fetched_at
		FROM company_info_cache
		WHERE razao_social ILIKE $1 OR nome_fantasia ILIKE $1
		ORDER BY razao_social ASC
		LIMIT $2
	`

	searchPattern := "%" + query + "%"
	rows, err := GetExecutor(ctx, r.pool).Query(ctx, sqlQuery, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search company info: %w", err)
	}
	defer rows.Close()

	var results []*company.CompanyInfo
	for rows.Next() {
		info, err := r.scanCompanyInfoRows(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, info)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search results: %w", err)
	}

	return results, nil
}

func (r *CompanyInfoRepository) scanCompanyInfo(row pgx.Row) (*company.CompanyInfo, error) {
	ci := &company.CompanyInfo{}

	var nomeFantasia, naturezaJuridica, logradouro, numero, complemento, bairro, cidade, uf, cep, cnaePrincipal *string
	var situacao string
	var dataAbertura time.Time
	var telefonesData []byte
	var emailsData []byte

	err := row.Scan(
		&ci.CNPJ,
		&ci.RazaoSocial,
		&nomeFantasia,
		&situacao,
		&naturezaJuridica,
		&dataAbertura,
		&logradouro,
		&numero,
		&complemento,
		&bairro,
		&cidade,
		&uf,
		&cep,
		&telefonesData,
		&emailsData,
		&cnaePrincipal,
		&ci.LastFetchedAt,
	)

	if err != nil {
		return nil, err
	}

	ci.Situacao = company.CadastralSituation(situacao)
	ci.DataAbertura = dataAbertura

	if nomeFantasia != nil {
		ci.NomeFantasia = *nomeFantasia
	}
	if naturezaJuridica != nil {
		ci.NaturezaJuridica = *naturezaJuridica
	}
	if logradouro != nil {
		ci.Endereco.Logradouro = *logradouro
	}
	if numero != nil {
		ci.Endereco.Numero = *numero
	}
	if complemento != nil {
		ci.Endereco.Complemento = *complemento
	}
	if bairro != nil {
		ci.Endereco.Bairro = *bairro
	}
	if cidade != nil {
		ci.Endereco.Cidade = *cidade
	}
	if uf != nil {
		ci.Endereco.UF = *uf
	}
	if cep != nil {
		ci.Endereco.CEP = *cep
	}
	if cnaePrincipal != nil {
		ci.CNAEPrincipal = *cnaePrincipal
	}

	if len(telefonesData) > 0 {
		_ = json.Unmarshal(telefonesData, &ci.Telefones)
	}
	if len(emailsData) > 0 {
		_ = json.Unmarshal(emailsData, &ci.Emails)
	}

	return ci, nil
}

func (r *CompanyInfoRepository) scanCompanyInfoRows(rows pgx.Rows) (*company.CompanyInfo, error) {
	return r.scanCompanyInfo(rows)
}
