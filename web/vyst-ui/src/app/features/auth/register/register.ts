import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, RouterModule } from '@angular/router';
import { AuthService } from '../../../core/services/auth';
import { TurnstileCaptchaComponent } from '../../../core/captcha';

@Component({
  selector: 'app-register',
  imports: [CommonModule, FormsModule, RouterModule, TurnstileCaptchaComponent],
  template: `
    <div class="min-h-screen flex">
      <!-- Left Side - Form -->
      <div class="flex-1 flex flex-col justify-center py-12 px-4 sm:px-6 lg:flex-none lg:px-20 xl:px-24 bg-white z-10">
        <div class="mx-auto w-full max-w-sm lg:w-96">
          <div class="text-center lg:text-left">
             <div class="flex items-center justify-center lg:justify-start mb-8">
                <div class="h-10 w-10 rounded-lg bg-gradient-to-br from-primary to-purple-600 flex items-center justify-center text-white font-bold text-xl shadow-lg">V</div>
                <span class="ml-3 text-2xl font-bold text-gray-900 tracking-tight">Vyst Identity</span>
             </div>
            <h2 class="mt-6 text-3xl font-extrabold text-gray-900 tracking-tight">Create your account</h2>
            <p class="mt-2 text-sm text-gray-600">
              Start your 14-day free trial. No credit card required.
            </p>
          </div>

          <div class="mt-8">
            <form class="space-y-6" (ngSubmit)="onSubmit()">
              <div>
                <label for="email" class="block text-sm font-medium text-gray-700">Email address</label>
                <div class="mt-1">
                  <input id="email" name="email" type="email" autocomplete="email" required [(ngModel)]="email"
                    class="appearance-none block w-full px-3 py-3 border border-gray-300 rounded-lg shadow-sm placeholder-gray-400 focus:outline-none focus:ring-primary focus:border-primary sm:text-sm transition duration-150 ease-in-out">
                </div>
              </div>

              <div class="space-y-1">
                <label for="password" class="block text-sm font-medium text-gray-700">Password</label>
                <div class="mt-1">
                  <input id="password" name="password" type="password" autocomplete="new-password" required [(ngModel)]="password"
                    class="appearance-none block w-full px-3 py-3 border border-gray-300 rounded-lg shadow-sm placeholder-gray-400 focus:outline-none focus:ring-primary focus:border-primary sm:text-sm transition duration-150 ease-in-out">
                </div>
                <p class="mt-1 text-xs text-gray-500">Must be at least 8 characters.</p>
              </div>

              <div>
                <label for="tenant-name" class="block text-sm font-medium text-gray-700">Company Name</label>
                <div class="mt-1">
                  <input id="tenant-name" name="tenantName" type="text" required [(ngModel)]="tenantName"
                    class="appearance-none block w-full px-3 py-3 border border-gray-300 rounded-lg shadow-sm placeholder-gray-400 focus:outline-none focus:ring-primary focus:border-primary sm:text-sm transition duration-150 ease-in-out">
                </div>
              </div>

              <div class="flex items-center">
                <input id="terms" name="terms" type="checkbox" required class="h-4 w-4 text-primary focus:ring-primary border-gray-300 rounded">
                <label for="terms" class="ml-2 block text-sm text-gray-900">
                  I agree to the <a href="#" class="text-primary hover:text-opacity-80">Terms</a> and <a href="#" class="text-primary hover:text-opacity-80">Privacy Policy</a>
                </label>
              </div>

              <!-- CAPTCHA -->
              <div class="mt-4">
                <app-turnstile-captcha (tokenChange)="captchaToken = $event" />
              </div>

              <div>
                <button type="submit" [disabled]="loading"
                  class="w-full flex justify-center py-3 px-4 border border-transparent rounded-lg shadow-sm text-sm font-medium text-white bg-primary hover:bg-opacity-90 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary disabled:opacity-50 disabled:cursor-not-allowed transition-all duration-200 transform hover:-translate-y-0.5">
                  @if (loading) {
                    <span class="flex items-center">
                      <svg class="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                      </svg>
                      Creating account...
                    </span>
                  } @else {
                    <span>Create account</span>
                  }
                </button>
              </div>
            </form>
            
            <p class="mt-8 text-center text-sm text-gray-600">
               Already have an account? 
               <a routerLink="/login" class="font-medium text-primary hover:text-opacity-80 transition-colors">Sign in</a>
            </p>
          </div>
        </div>
      </div>

      <!-- Right Side - Image/Gradient -->
      <div class="hidden lg:block relative w-0 flex-1 overflow-hidden">
        <div class="absolute inset-0 bg-gradient-to-bl from-gray-900 to-gray-800">
           <div class="absolute inset-0 bg-primary opacity-20 mix-blend-overlay"></div>
           <div class="absolute inset-0 flex items-center justify-center">
              <div class="text-center px-8">
                 <h2 class="text-4xl font-bold text-white mb-4">Join the Future of Identity</h2>
                 <p class="text-lg text-gray-300 max-w-md mx-auto">Scalable, secure, and developer-friendly authentication infrastructure.</p>
              </div>
           </div>
           <!-- Decorative circles -->
           <div class="absolute top-0 left-0 -ml-20 -mt-20 w-96 h-96 rounded-full bg-primary opacity-10 blur-3xl"></div>
           <div class="absolute bottom-0 right-0 -mr-20 -mb-20 w-96 h-96 rounded-full bg-purple-500 opacity-10 blur-3xl"></div>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .bg-primary { background-color: var(--primary-color, #4f46e5); }
    .text-primary { color: var(--primary-color, #4f46e5); }
    .focus\\:ring-primary:focus { --tw-ring-color: var(--primary-color, #4f46e5); }
    .focus\\:border-primary:focus { border-color: var(--primary-color, #4f46e5); }
    .from-primary { --tw-gradient-from: var(--primary-color, #4f46e5); }
  `]
})
export class RegisterComponent {
  email = '';
  password = '';
  tenantName = '';
  captchaToken = '';
  loading = false;

  private authService = inject(AuthService);
  private router = inject(Router);

  onSubmit() {
    this.loading = true;

    const data = {
      email: this.email,
      password: this.password,
      tenant_name: this.tenantName
    };

    this.authService.register(data, this.captchaToken).subscribe({
      next: () => {
        this.router.navigate(['/login'], { queryParams: { registered: 'true' } });
      },
      error: () => {
        // Handled by global error interceptor
        this.loading = false;
      }
    });
  }
}
