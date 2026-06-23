import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router, RouterModule } from '@angular/router';
import { CompanyService, Company } from '../../../core/services/company.service';
import { AuthService } from '../../../core/services/auth';

@Component({
  selector: 'app-company-selection',
  standalone: true,
  imports: [CommonModule, RouterModule],
  template: `
    <div class="min-h-screen bg-gray-50 flex flex-col justify-center py-12 sm:px-6 lg:px-8">
      <div class="sm:mx-auto sm:w-full sm:max-w-md">
        <h2 class="mt-6 text-center text-3xl font-extrabold text-gray-900">
          Select Company
        </h2>
        <p class="mt-2 text-center text-sm text-gray-600">
          Choose a company to access or create a new one
        </p>
      </div>

      <div class="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
        <div class="bg-white py-8 px-4 shadow sm:rounded-lg sm:px-10">
          
          <div *ngIf="loading()" class="text-center py-4">
            Loading companies...
          </div>

          <div *ngIf="error()" class="mb-4 bg-red-50 border border-red-200 text-red-600 px-4 py-3 rounded relative" role="alert">
            <span class="block sm:inline">{{ error() }}</span>
          </div>

          <ul *ngIf="!loading() && companies().length > 0" class="divide-y divide-gray-200">
            <li *ngFor="let company of companies()" class="py-4 flex justify-between items-center cursor-pointer hover:bg-gray-50 px-2 rounded" (click)="selectCompany(company)">
              <div>
                <p class="text-sm font-medium text-gray-900">{{ company.nome_fantasia || company.razao_social }}</p>
                <p class="text-sm text-gray-500">{{ company.cnpj }}</p>
              </div>
              <button class="bg-indigo-600 text-white px-3 py-1 rounded text-sm hover:bg-indigo-700">
                Select
              </button>
            </li>
          </ul>

          <div *ngIf="!loading() && companies().length === 0" class="text-center py-4 text-gray-500">
            No companies found. Create one to get started.
          </div>

          <div class="mt-6">
            <button (click)="createCompany()" class="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-indigo-600 bg-white border-indigo-600 hover:bg-indigo-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500">
              Register New Company
            </button>
          </div>
          
           <div class="mt-4 text-center">
            <span class="text-sm text-gray-500 cursor-pointer hover:text-gray-900" (click)="logout()">Sign Out</span>
          </div>

        </div>
      </div>
    </div>
  `
})
export class CompanySelectionComponent implements OnInit {
  private companyService = inject(CompanyService);
  private authService = inject(AuthService);
  private router = inject(Router);

  companies = signal<Company[]>([]);
  loading = signal<boolean>(false);
  error = signal<string | null>(null);

  ngOnInit() {
    this.loadCompanies();
  }

  loadCompanies() {
    this.loading.set(true);
    this.companyService.getCompanies().subscribe({
      next: (data) => {
        this.companies.set(data);
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set('Failed to load companies');
        this.loading.set(false);
        console.error(err);
      }
    });
  }

  selectCompany(company: Company) {
    this.loading.set(true);
    this.authService.switchCompany(company.id).subscribe({
      next: () => {
        this.loading.set(false);
        // Navigation is handled in AuthService
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to switch company');
        this.loading.set(false);
      }
    });
  }
  
  createCompany() {
      // TODO: Navigate to create company page or open modal
      // this.router.navigate(['/auth/company/create']);
      alert('Create Company Not Implemented yet for UI');
  }

  logout() {
      this.authService.logout();
  }
}
