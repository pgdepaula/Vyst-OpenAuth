import { Component, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { AppConfigService } from '../../../core/services/app-config.service';

@Component({
  selector: 'app-setup-organization',
  imports: [CommonModule, ReactiveFormsModule],
  template: `
    <div class="flex min-h-full flex-col justify-center py-12 sm:px-6 lg:px-8">
      <div class="sm:mx-auto sm:w-full sm:max-w-md">
        <h2 class="mt-6 text-center text-3xl font-bold tracking-tight text-gray-900">Setup your Organization</h2>
        <p class="mt-2 text-center text-sm text-gray-600">
          Create a new organization to get started with Vyst Identity.
        </p>
      </div>

      <div class="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
        <div class="bg-white py-8 px-4 shadow sm:rounded-lg sm:px-10">
          <form [formGroup]="form" (ngSubmit)="onSubmit()" class="space-y-6">
            <div>
              <label for="name" class="block text-sm font-medium text-gray-700">Organization Name</label>
              <div class="mt-1">
                <input id="name" name="name" type="text" formControlName="name" required class="block w-full appearance-none rounded-md border border-gray-300 px-3 py-2 placeholder-gray-400 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-indigo-500 sm:text-sm">
                @if (form.get('name')?.touched && form.get('name')?.invalid) {
                  <p class="mt-2 text-sm text-red-600">Organization name is required.</p>
                }
              </div>
            </div>

            <div>
              <button type="submit" [disabled]="form.invalid || isLoading()" class="flex w-full justify-center rounded-md border border-transparent bg-indigo-600 py-2 px-4 text-sm font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 disabled:opacity-50">
                {{ isLoading() ? 'Creating...' : 'Create Organization' }}
              </button>
            </div>
            
            @if (errorMessage()) {
              <div class="rounded-md bg-red-50 p-4">
                <div class="flex">
                  <div class="ml-3">
                    <h3 class="text-sm font-medium text-red-800">{{ errorMessage() }}</h3>
                  </div>
                </div>
              </div>
            }
          </form>
        </div>
      </div>
    </div>
  `
})
export class SetupOrganizationComponent {
  private fb = inject(FormBuilder);
  private http = inject(HttpClient);
  private router = inject(Router);
  private configService = inject(AppConfigService);

  isLoading = signal(false);
  errorMessage = signal('');

  form = this.fb.group({
    name: ['', Validators.required]
  });

  onSubmit() {
    if (this.form.invalid) return;

    this.isLoading.set(true);
    this.errorMessage.set('');

    const url = `${this.configService.apiUrl}/api/v1/tenants`;
    this.http.post(url, this.form.value).subscribe({
      next: () => {
        this.isLoading.set(false);
        this.router.navigate(['/admin/dashboard']);
      },
      error: (err) => {
        this.isLoading.set(false);
        this.errorMessage.set('Failed to create organization. Please try again.');
        console.error('Failed to create tenant', err);
      }
    });
  }
}
