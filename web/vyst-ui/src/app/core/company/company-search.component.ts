import { Component, input, output, signal, computed, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormControl } from '@angular/forms';
import { catchError, debounceTime, distinctUntilChanged, of, switchMap, tap } from 'rxjs';
import { HttpClient } from '@angular/common/http';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';

export interface CompanyPreview {
  cnpj: string;
  razao_social: string;
  nome_fantasia?: string;
  situacao: string;
  cnae_principal?: string;
}

export interface CompanyLookupResponse {
  items: CompanyPreview[];
}

@Component({
  selector: 'vyst-company-search',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule],
  template: `
    <div class="company-search-container">
      <div class="input-wrapper">
        <input 
          type="text" 
          [formControl]="searchControl" 
          [placeholder]="placeholder()"
          class="search-input"
          [class.loading]="isLoading()"
          (focus)="isFocused.set(true)"
          (blur)="onBlur()"
          aria-label="Buscar empresa por CNPJ ou nome"
        />
        @if(isLoading()) {
          <div class="spinner"></div>
        }
      </div>

      @if(isFocused() && (results().length > 0 || hasError() || searchQuery().length >= 3)) {
        <div class="dropdown-results">
          @if(hasError()) {
            <div class="error-msg">Erro ao buscar empresas. Tente novamente mais tarde.</div>
          } @else if(results().length === 0 && !isLoading() && searchQuery().length >= 3) {
            <div class="empty-msg">Nenhuma empresa encontrada com este termo.</div>
          } @else {
            @for(company of results(); track company.cnpj) {
              <div 
                class="result-item" 
                (click)="selectCompany(company)"
                (mousedown)="$event.preventDefault()"
              >
                <div class="company-names">
                  <span class="razao" [innerHTML]="highlightMatch(company.razao_social, searchControl.value!)"></span>
                  @if(company.nome_fantasia) {
                    <span class="fantasia" [innerHTML]="highlightMatch(company.nome_fantasia, searchControl.value!)"></span>
                  }
                </div>
                <div class="company-meta">
                  <span class="cnpj">{{ formatCnpj(company.cnpj) }}</span>
                  <span class="status" [class.active]="company.situacao === 'ATIVA'">{{ company.situacao }}</span>
                </div>
              </div>
            }
          }
        </div>
      }
    </div>
  `,
  styles: [`
    .company-search-container {
      position: relative;
      width: 100%;
      font-family: inherit;
    }
    .input-wrapper {
      position: relative;
    }
    .search-input {
      width: 100%;
      padding: 12px 16px;
      padding-right: 40px;
      border: 1px solid #ccc;
      border-radius: 8px;
      font-size: 1rem;
      outline: none;
      transition: border-color 0.2s, box-shadow 0.2s;
    }
    .search-input:focus {
      border-color: #0066cc;
      box-shadow: 0 0 0 3px rgba(0, 102, 204, 0.2);
    }
    .spinner {
      position: absolute;
      right: 12px;
      top: 50%;
      transform: translateY(-50%);
      width: 20px;
      height: 20px;
      border: 2px solid #ccc;
      border-top-color: #0066cc;
      border-radius: 50%;
      animation: spin 1s linear infinite;
    }
    @keyframes spin { 
      to { transform: translateY(-50%) rotate(360deg); } 
    }
    .dropdown-results {
      position: absolute;
      top: calc(100% + 4px);
      left: 0;
      width: 100%;
      background: white;
      border: 1px solid #eee;
      border-radius: 8px;
      box-shadow: 0 4px 12px rgba(0,0,0,0.1);
      max-height: 300px;
      overflow-y: auto;
      z-index: 1000;
    }
    .result-item {
      padding: 12px 16px;
      border-bottom: 1px solid #f5f5f5;
      cursor: pointer;
      transition: background-color 0.2s;
    }
    .result-item:hover {
      background-color: #f9f9f9;
    }
    .result-item:last-child {
      border-bottom: none;
    }
    .company-names {
      display: flex;
      flex-direction: column;
      gap: 4px;
      margin-bottom: 6px;
    }
    .razao {
      font-weight: 600;
      color: #333;
    }
    .fantasia {
      font-size: 0.85rem;
      color: #666;
    }
    .company-meta {
      display: flex;
      justify-content: space-between;
      align-items: center;
      font-size: 0.85rem;
    }
    .cnpj {
      color: #666;
      font-family: monospace;
    }
    .status {
      padding: 2px 8px;
      border-radius: 12px;
      background: #eee;
      color: #555;
      font-size: 0.75rem;
      font-weight: 500;
    }
    .status.active {
      background: #e6f4ea;
      color: #1e8e3e;
    }
    .empty-msg, .error-msg {
      padding: 16px;
      text-align: center;
      color: #666;
      font-size: 0.9rem;
    }
    .error-msg {
      color: #d93025;
    }
    .highlight {
      background-color: #fef08a;
      font-weight: bold;
    }
  `]
})
export class CompanySearchComponent {
  private http = inject(HttpClient);

  // Inputs
  placeholder = input<string>('Buscar empresa... (mínimo 3 caracteres ou CNPJ)');
  
  // Outputs
  companySelected = output<CompanyPreview>();

  // State
  searchControl = new FormControl('');
  searchQuery = signal('');
  results = signal<CompanyPreview[]>([]);
  isLoading = signal(false);
  hasError = signal(false);
  isFocused = signal(false);

  constructor() {
    this.searchControl.valueChanges.pipe(
      takeUntilDestroyed(),
      debounceTime(300),
      distinctUntilChanged(),
      tap(query => {
        this.searchQuery.set(query ?? '');
        this.isLoading.set(true);
        this.hasError.set(false);
      }),
      switchMap(query => {
        if (!query || query.length < 3) {
          this.isLoading.set(false);
          return of({ items: [] } as CompanyLookupResponse);
        }
        return this.http.get<CompanyLookupResponse>(`/api/v1/companies/lookup?q=${encodeURIComponent(query)}`).pipe(
          catchError(() => {
            this.hasError.set(true);
            return of({ items: [] } as CompanyLookupResponse);
          })
        );
      })
    ).subscribe(response => {
      this.isLoading.set(false);
      if (!this.hasError() && response?.items) {
        this.results.set(response.items);
      } else {
        this.results.set([]);
      }
    });
  }

  selectCompany(company: CompanyPreview) {
    this.companySelected.emit(company);
    this.searchControl.setValue(company.razao_social, { emitEvent: false });
    this.isFocused.set(false);
  }

  onBlur() {
    // Small timeout to allow click event on result item to fire before hiding dropdown
    setTimeout(() => {
      this.isFocused.set(false);
    }, 200);
  }

  formatCnpj(cnpj: string): string {
    if (!cnpj || cnpj.length !== 14) return cnpj;
    return cnpj.replace(/^(\d{2})(\d{3})(\d{3})(\d{4})(\d{2})$/, "$1.$2.$3/$4-$5");
  }

  highlightMatch(text: string, query: string): string {
    if (!query || !text) return text;
    
    // Escape regex characters
    const escapedQuery = query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    const regex = new RegExp(`(${escapedQuery})`, 'gi');
    
    // Replace using an implicit inline style wrapper (safely as we use it with innerHTML)
    return text.replace(regex, '<span style="background-color: #fce83a; font-weight: bold;">$1</span>');
  }
}
