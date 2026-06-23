import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { AppConfigService } from '../../../../core/services/app-config.service';

interface Tenant {
  id: string;
  name: string;
  status: string;
  created_at: string;
}

@Component({
  selector: 'app-tenants-list',
  imports: [CommonModule],
  template: `
    <div class="px-4 sm:px-6 lg:px-8 py-8">
      <div class="sm:flex sm:items-center">
        <div class="sm:flex-auto">
          <h1 class="text-xl font-semibold text-gray-900">Tenants</h1>
          <p class="mt-2 text-sm text-gray-700">
            A list of all tenants in the system.
          </p>
        </div>
      </div>
      
      <div class="mt-8 flex flex-col">
        <div class="-my-2 -mx-4 overflow-x-auto sm:-mx-6 lg:-mx-8">
          <div class="inline-block min-w-full py-2 align-middle md:px-6 lg:px-8">
            <div class="overflow-hidden shadow ring-1 ring-black ring-opacity-5 md:rounded-lg">
              <table class="min-w-full divide-y divide-gray-300">
                <thead class="bg-gray-50">
                  <tr>
                    <th scope="col" class="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 sm:pl-6">Name</th>
                    <th scope="col" class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900">Status</th>
                    <th scope="col" class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900">Created At</th>
                    <th scope="col" class="relative py-3.5 pl-3 pr-4 sm:pr-6">
                      <span class="sr-only">Actions</span>
                    </th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-gray-200 bg-white">
                  @for (tenant of tenants(); track tenant.id) {
                    <tr>
                      <td class="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 sm:pl-6">{{ tenant.name }}</td>
                      <td class="whitespace-nowrap px-3 py-4 text-sm">
                        <span class="inline-flex rounded-full px-2 text-xs font-semibold leading-5"
                              [class.bg-green-100]="tenant.status === 'active'"
                              [class.text-green-800]="tenant.status === 'active'"
                              [class.bg-red-100]="tenant.status === 'suspended'"
                              [class.text-red-800]="tenant.status === 'suspended'">
                          {{ tenant.status }}
                        </span>
                      </td>
                      <td class="whitespace-nowrap px-3 py-4 text-sm text-gray-500">{{ tenant.created_at | date }}</td>
                      <td class="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6">
                        @if (tenant.status === 'active') {
                          <button (click)="suspendTenant(tenant.id)" class="text-red-600 hover:text-red-900">Suspend</button>
                        }
                      </td>
                    </tr>
                  } @empty {
                    <tr>
                      <td colspan="4" class="px-6 py-4 text-center text-sm text-gray-500">
                        No tenants found.
                      </td>
                    </tr>
                  }
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
    </div>
  `
})
export class TenantsListComponent implements OnInit {
  private http = inject(HttpClient);
  private configService = inject(AppConfigService);

  tenants = signal<Tenant[]>([]);

  ngOnInit() {
    this.loadTenants();
  }

  loadTenants() {
    const url = `${this.configService.apiUrl}/api/v1/admin/tenants`;
    this.http.get<Tenant[]>(url).subscribe({
      next: (data) => this.tenants.set(data || []),
      error: (err) => console.error('Failed to load tenants', err)
    });
  }

  suspendTenant(id: string) {
    if (!confirm('Are you sure you want to suspend this tenant?')) return;

    const url = `${this.configService.apiUrl}/api/v1/admin/tenants/${id}/suspend`;
    this.http.post(url, {}).subscribe({
      next: () => this.loadTenants(),
      error: (err) => console.error('Failed to suspend tenant', err)
    });
  }
}
