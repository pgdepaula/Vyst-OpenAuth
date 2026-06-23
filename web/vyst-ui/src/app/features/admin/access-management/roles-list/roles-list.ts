import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { AppConfigService } from '../../../../core/services/app-config.service';

interface Role {
  id: string;
  name: string;
  description: string;
  permissions: string[];
  created_at: string;
}

@Component({
  selector: 'app-roles-list',
  imports: [CommonModule, RouterLink],
  template: `
    <div class="px-4 sm:px-6 lg:px-8 py-8">
      <div class="sm:flex sm:items-center">
        <div class="sm:flex-auto">
          <h1 class="text-xl font-semibold text-gray-900">Roles & Permissions</h1>
          <p class="mt-2 text-sm text-gray-700">
            A list of all roles in your organization including their name, description, and assigned permissions.
          </p>
        </div>
        <div class="mt-4 sm:mt-0 sm:ml-16 sm:flex-none">
          <a routerLink="/admin/roles/new" class="inline-flex items-center justify-center rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 sm:w-auto">
            Add Role
          </a>
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
                    <th scope="col" class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900">Description</th>
                    <th scope="col" class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900">Permissions</th>
                    <th scope="col" class="relative py-3.5 pl-3 pr-4 sm:pr-6">
                      <span class="sr-only">Edit</span>
                    </th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-gray-200 bg-white">
                  @for (role of roles(); track role.id) {
                    <tr>
                      <td class="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 sm:pl-6">{{ role.name }}</td>
                      <td class="whitespace-nowrap px-3 py-4 text-sm text-gray-500">{{ role.description }}</td>
                      <td class="px-3 py-4 text-sm text-gray-500">
                        <div class="flex flex-wrap gap-1">
                          @for (perm of role.permissions.slice(0, 3); track perm) {
                            <span class="inline-flex items-center rounded-full bg-blue-100 px-2.5 py-0.5 text-xs font-medium text-blue-800">
                              {{ perm }}
                            </span>
                          }
                          @if (role.permissions.length > 3) {
                            <span class="inline-flex items-center rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-800">
                              +{{ role.permissions.length - 3 }} more
                            </span>
                          }
                        </div>
                      </td>
                      <td class="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6">
                        <a [routerLink]="['/admin/roles', role.id]" class="text-indigo-600 hover:text-indigo-900">Edit<span class="sr-only">, {{ role.name }}</span></a>
                      </td>
                    </tr>
                  } @empty {
                    <tr>
                      <td colspan="4" class="px-6 py-4 text-center text-sm text-gray-500">
                        No roles found. Create one to get started.
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
export class RolesListComponent implements OnInit {
  private http = inject(HttpClient);
  private configService = inject(AppConfigService);

  roles = signal<Role[]>([]);

  ngOnInit() {
    this.loadRoles();
  }

  loadRoles() {
    const url = `${this.configService.apiUrl}/api/v1/roles`;
    this.http.get<Role[]>(url).subscribe({
      next: (data) => this.roles.set(data || []),
      error: (err) => console.error('Failed to load roles', err)
    });
  }
}
