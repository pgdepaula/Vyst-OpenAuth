import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { AuthService } from '../../../core/services/auth';

@Component({
  selector: 'app-reset-password',
  imports: [CommonModule, ReactiveFormsModule, RouterLink],
  template: `
    <div class="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
      <div class="max-w-md w-full space-y-8">
        <div>
          <h2 class="mt-6 text-center text-3xl font-extrabold text-gray-900">
            Reset your password
          </h2>
          <p class="mt-2 text-center text-sm text-gray-600">
            Enter your new password below.
          </p>
        </div>

        @if (errorMessage()) {
          <div class="rounded-md bg-red-50 p-4">
            <div class="flex">
              <div class="flex-shrink-0">
                <!-- Heroicon name: solid/x-circle -->
                <svg class="h-5 w-5 text-red-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                  <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
                </svg>
              </div>
              <div class="ml-3">
                <h3 class="text-sm font-medium text-red-800">
                  {{ errorMessage() }}
                </h3>
              </div>
            </div>
          </div>
        }

        @if (successMessage()) {
           <div class="rounded-md bg-green-50 p-4">
            <div class="flex">
              <div class="flex-shrink-0">
                <!-- Heroicon name: solid/check-circle -->
                <svg class="h-5 w-5 text-green-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                  <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
                </svg>
              </div>
              <div class="ml-3">
                <p class="text-sm font-medium text-green-800">
                  {{ successMessage() }}
                </p>
                <p class="mt-2 text-sm text-green-700">
                  <a routerLink="/login" class="font-medium underline hover:text-green-600">
                    Click here to sign in
                  </a>
                </p>
              </div>
            </div>
          </div>
        } @else {
          <form class="mt-8 space-y-6" [formGroup]="form" (ngSubmit)="onSubmit()">
            <input type="hidden" name="remember" value="true">
            <div class="rounded-md shadow-sm -space-y-px">
              <div>
                <label for="password" class="sr-only">New Password</label>
                <input id="password" name="password" type="password" autocomplete="new-password" required class="appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-t-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm" placeholder="New Password" formControlName="password">
              </div>
              <div>
                <label for="confirm-password" class="sr-only">Confirm Password</label>
                <input id="confirm-password" name="confirmPassword" type="password" autocomplete="new-password" required class="appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-b-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm" placeholder="Confirm Password" formControlName="confirmPassword">
              </div>
            </div>

            @if (form.hasError('mismatch') && form.get('confirmPassword')?.touched) {
               <p class="text-sm text-red-600">Passwords do not match.</p>
            }

            <div>
              <button type="submit" [disabled]="form.invalid || isLoading()" class="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50">
                <span class="absolute left-0 inset-y-0 flex items-center pl-3">
                  <!-- Heroicon name: solid/lock-closed -->
                  <svg class="h-5 w-5 text-indigo-500 group-hover:text-indigo-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                    <path fill-rule="evenodd" d="M5 9V7a5 5 0 0110 0v2a2 2 0 012 2v5a2 2 0 01-2 2H5a2 2 0 01-2-2v-5a2 2 0 012-2zm8-2v2H7V7a3 3 0 016 0z" clip-rule="evenodd" />
                  </svg>
                </span>
                {{ isLoading() ? 'Resetting...' : 'Reset Password' }}
              </button>
            </div>
          </form>
        }
      </div>
    </div>
  `
})
export class ResetPasswordComponent implements OnInit {
  private fb = inject(FormBuilder);
  private authService = inject(AuthService);
  private route = inject(ActivatedRoute);
  private router = inject(Router);

  token = signal('');
  isLoading = signal(false);
  successMessage = signal('');
  errorMessage = signal('');

  form = this.fb.group({
    password: ['', [Validators.required, Validators.minLength(8)]],
    confirmPassword: ['', [Validators.required]]
  }, { validators: this.passwordMatchValidator });

  ngOnInit() {
    this.route.queryParams.subscribe(params => {
      this.token.set(params['token'] || '');
      if (!this.token()) {
        this.errorMessage.set('Invalid or missing reset token.');
      }
    });
  }

  passwordMatchValidator(g: any) {
    return g.get('password').value === g.get('confirmPassword').value
      ? null : { 'mismatch': true };
  }

  onSubmit() {
    if (this.form.invalid || !this.token()) return;

    this.isLoading.set(true);
    this.errorMessage.set('');

    const password = this.form.get('password')?.value;

    if (password) {
      this.authService.resetPassword(this.token(), password).subscribe({
        next: () => {
          this.isLoading.set(false);
          this.successMessage.set('Your password has been successfully reset.');
          // Optionally redirect after a delay
          // setTimeout(() => this.router.navigate(['/login']), 3000);
        },
        error: (err) => {
          this.isLoading.set(false);
          this.errorMessage.set('Failed to reset password. The link may have expired.');
          console.error('Reset password error:', err);
        }
      });
    }
  }
}
