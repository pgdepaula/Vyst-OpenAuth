import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { AppConfigService } from './app-config.service';

export interface Company {
  id: string;
  tenant_id: string;
  cnpj: string;
  razao_social: string;
  nome_fantasia: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface CreateCompanyRequest {
  cnpj: string;
  razao_social: string;
  nome_fantasia: string;
  endereco?: {
    logradouro: string;
    numero: string;
    complemento?: string;
    bairro: string;
    cidade: string;
    uf: string;
    cep: string;
  };
}

@Injectable({
  providedIn: 'root',
})
export class CompanyService {
  private http = inject(HttpClient);
  private configService = inject(AppConfigService);

  getCompanies(): Observable<Company[]> {
    return this.http.get<Company[]>(`${this.configService.apiUrl}/api/v1/companies`);
  }

  createCompany(data: CreateCompanyRequest): Observable<Company> {
    return this.http.post<Company>(`${this.configService.apiUrl}/api/v1/companies`, data);
  }

  getCompany(id: string): Observable<Company> {
    return this.http.get<Company>(`${this.configService.apiUrl}/api/v1/companies/${id}`);
  }
}
