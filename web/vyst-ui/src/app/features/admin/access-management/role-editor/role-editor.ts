import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, Validators, FormArray } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { AppConfigService } from '../../../../core/services/app-config.service';

@Component({
  selector: 'app-role-editor',
  imports: [CommonModule, ReactiveFormsModule, RouterLink],
  template: `
    <div class="px-4 sm:px-6 lg:px-8 py-8">
      <div class="md:flex md:items-center md:justify-between">
        <div class="min-w-0 flex-1">
          <h2 class="text-2xl font-bold leading-7 text-gray-900 sm:truncate sm:text-3xl sm:tracking-tight">
            {{ isEditMode() ? 'Edit Role' : 'Create Role' }}
          </h2>
        </div>
      </div>

      <form [formGroup]="form" (ngSubmit)="onSubmit()" class="mt-8 space-y-8 divide-y divide-gray-200">
        <div class="space-y-8 divide-y divide-gray-200 sm:space-y-5">
          <div class="space-y-6 sm:space-y-5">
            <div>
              <h3 class="text-lg font-medium leading-6 text-gray-900">Role Details</h3>
              <p class="mt-1 max-w-2xl text-sm text-gray-500">
                Define the role name and its description.
              </p>
            </div>

            <div class="space-y-6 sm:space-y-5">
              <div class="sm:grid sm:grid-cols-3 sm:items-start sm:gap-4 sm:border-t sm:border-gray-200 sm:pt-5">
                <label for="name" class="block text-sm font-medium text-gray-700 sm:mt-px sm:pt-2">Role Name</label>
                <div class="mt-1 sm:col-span-2 sm:mt-0">
                  <input type="text" name="name" id="name" formControlName="name" class="block w-full max-w-lg rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:max-w-xs sm:text-sm">
                  @if (form.get('name')?.touched && form.get('name')?.invalid) {
                    <p class="mt-2 text-sm text-red-600">Name is required.</p>
                  }
                </div>
              </div>

              <div class="sm:grid sm:grid-cols-3 sm:items-start sm:gap-4 sm:border-t sm:border-gray-200 sm:pt-5">
                <label for="description" class="block text-sm font-medium text-gray-700 sm:mt-px sm:pt-2">Description</label>
                <div class="mt-1 sm:col-span-2 sm:mt-0">
                  <textarea id="description" name="description" rows="3" formControlName="description" class="block w-full max-w-lg rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"></textarea>
                  <p class="mt-2 text-sm text-gray-500">Brief description of what this role allows.</p>
                </div>
              </div>
            </div>
          </div>

          <div class="pt-8 space-y-6 sm:pt-10 sm:space-y-5">
            <div>
              <h3 class="text-lg font-medium leading-6 text-gray-900">Permissions</h3>
              <p class="mt-1 max-w-2xl text-sm text-gray-500">
                Select the permissions assigned to this role.
              </p>
            </div>

            <div class="space-y-6 sm:space-y-5">
              <fieldset>
                <legend class="sr-only">Permissions</legend>
                <div class="divide-y divide-gray-200 border-b border-t border-gray-200">
                  @for (group of permissionGroups; track group.name) {
                    <div class="relative flex items-start py-4">
                      <div class="min-w-0 flex-1 text-sm">
                        <label class="font-medium text-gray-700 select-none">{{ group.name }}</label>
                        <div class="mt-2 grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
                           @for (perm of group.permissions; track perm.value) {
                             <div class="relative flex items-start">
                                <div class="flex h-5 items-center">
                                  <input [id]="perm.value" [value]="perm.value" type="checkbox" 
                                         [checked]="hasPermission(perm.value)"
                                         (change)="onPermissionChange($event, perm.value)"
                                         class="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500">
                                </div>
                                <div class="ml-3 text-sm">
                                  <label [for]="perm.value" class="font-medium text-gray-700">{{ perm.label }}</label>
                                  <p class="text-gray-500">{{ perm.description }}</p>
                                </div>
                             </div>
                           }
                        </div>
                      </div>
                    </div>
                  }
                </div>
              </fieldset>
            </div>
          </div>
        </div>

        <div class="pt-5">
          <div class="flex justify-end gap-x-3">
            <a routerLink="/admin/roles" class="rounded-md border border-gray-300 bg-white py-2 px-4 text-sm font-medium text-gray-700 shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2">Cancel</a>
            <button type="submit" [disabled]="form.invalid || isLoading()" class="inline-flex justify-center rounded-md border border-transparent bg-indigo-600 py-2 px-4 text-sm font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 disabled:opacity-50">
              {{ isLoading() ? 'Saving...' : 'Save' }}
            </button>
          </div>
        </div>
      </form>
    </div>
  `
})
export class RoleEditorComponent implements OnInit {
  private fb = inject(FormBuilder);
  private http = inject(HttpClient);
  private router = inject(Router);
  private route = inject(ActivatedRoute);
  private configService = inject(AppConfigService);

  isEditMode = signal(false);
  isLoading = signal(false);
  roleId = signal<string | null>(null);

  form = this.fb.group({
    name: ['', Validators.required],
    description: [''],
    permissions: this.fb.array<string>([])
  });

  // Hardcoded permission definitions for now
  permissionGroups = [
    {
      name: 'Users',
      permissions: [
        { value: 'users:read', label: 'Read Users', description: 'View user list and details' },
        { value: 'users:write', label: 'Write Users', description: 'Create and update users' },
        { value: 'users:delete', label: 'Delete Users', description: 'Remove users from the system' },
      ]
    },
    {
      name: 'Roles',
      permissions: [
        { value: 'roles:read', label: 'Read Roles', description: 'View roles and permissions' },
        { value: 'roles:write', label: 'Write Roles', description: 'Create and update roles' },
        { value: 'roles:delete', label: 'Delete Roles', description: 'Remove roles' },
      ]
    },
    {
      name: 'Tenants',
      permissions: [
        { value: 'tenants:read', label: 'Read Tenant', description: 'View tenant settings' },
        { value: 'tenants:write', label: 'Write Tenant', description: 'Update tenant settings' },
      ]
    }
  ];

  ngOnInit() {
    this.route.params.subscribe(params => {
      if (params['id']) {
        this.isEditMode.set(true);
        this.roleId.set(params['id']);
        this.loadRole(params['id']);
      }
    });
  }

  loadRole(id: string) {
    const url = `${this.configService.apiUrl}/api/v1/roles/${id}`;
    this.http.get<any>(url).subscribe({
      next: (role) => {
        this.form.patchValue({
          name: role.name,
          description: role.description
        });

        // Clear and set permissions
        const permissionsArray = this.form.get('permissions') as FormArray;
        permissionsArray.clear();
        role.permissions.forEach((p: string) => permissionsArray.push(this.fb.control(p)));
      },
      error: (err) => console.error('Failed to load role', err)
    });
  }

  hasPermission(perm: string): boolean {
    const permissions = this.form.get('permissions')?.value as string[];
    return permissions.includes(perm);
  }

  onPermissionChange(event: any, perm: string) {
    const permissionsArray = this.form.get('permissions') as FormArray;
    if (event.target.checked) {
      permissionsArray.push(this.fb.control(perm));
    } else {
      const index = permissionsArray.controls.findIndex(x => x.value === perm);
      if (index >= 0) {
        permissionsArray.removeAt(index);
      }
    }
  }

  onSubmit() {
    if (this.form.invalid) return;

    this.isLoading.set(true);
    const roleData = this.form.value;

    if (this.isEditMode()) {
      const url = `${this.configService.apiUrl}/api/v1/roles/${this.roleId()}`;
      this.http.put(url, roleData).subscribe({
        next: () => {
          this.isLoading.set(false);
          this.router.navigate(['/admin/roles']);
        },
        error: (err) => {
          this.isLoading.set(false);
          console.error('Failed to update role', err);
        }
      });
    } else {
      const url = `${this.configService.apiUrl}/api/v1/roles`;
      this.http.post(url, roleData).subscribe({
        next: () => {
          this.isLoading.set(false);
          this.router.navigate(['/admin/roles']);
        },
        error: (err) => {
          this.isLoading.set(false);
          console.error('Failed to create role', err);
        }
      });
    }
  }
}
