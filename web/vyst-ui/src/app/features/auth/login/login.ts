import { Component, inject, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, RouterModule } from '@angular/router';
import { AuthService } from '../../../core/services/auth';
import { BrandingService, Theme } from '../../../core/services/branding';
import { ErrorHandlerService } from '../../../core/services/error-handler.service';
import { TurnstileCaptchaComponent } from '../../../core/captcha';

type LoginStep = 'credentials' | 'totp';

@Component({
  selector: 'app-login',
  imports: [CommonModule, FormsModule, RouterModule, TurnstileCaptchaComponent],
  template: `
    <div class="min-h-screen flex font-sans bg-gray-50">
      <!-- Left Side - Form -->
      <div class="flex-1 flex flex-col justify-center py-12 px-4 sm:px-6 lg:flex-none lg:px-20 xl:px-24 bg-white z-10 relative shadow-2xl animate-fade-in">
        <div class="mx-auto w-full max-w-sm lg:w-96">
          <div class="text-center lg:text-left">
             @if (theme?.logoUrl) {
               <div class="h-12 w-auto mb-10">
                  <img [src]="theme?.logoUrl" alt="Logo" class="h-12">
               </div>
             } @else {
               <div class="flex items-center justify-center lg:justify-start mb-10">
                  <div class="h-12 w-12 rounded-xl bg-gradient-to-br from-primary-600 to-secondary-500 flex items-center justify-center text-white font-bold text-2xl shadow-lg ring-4 ring-primary-50">V</div>
                  <span class="ml-4 text-3xl font-bold text-gray-900 tracking-tight">Vyst Identity</span>
               </div>
             }
            <h2 class="mt-6 text-3xl font-extrabold text-gray-900 tracking-tight">
              {{ step() === 'totp' ? 'Two-Factor Authentication' : 'Welcome back' }}
            </h2>
            <p class="mt-2 text-sm text-gray-600">
              {{ step() === 'totp' ? 'Enter the 6-digit code from your authenticator app.' : 'Please enter your details to sign in.' }}
            </p>
          </div>

          <div class="mt-10">
            <!-- Credentials Step -->
            @if (step() === 'credentials') {
              <form class="space-y-6" (ngSubmit)="onSubmit()">
                <div>
                  <label for="email" class="block text-sm font-medium text-gray-700">Email address</label>
                  <div class="mt-1">
                    <input id="email" name="email" type="email" autocomplete="email" required [(ngModel)]="email"
                      class="appearance-none block w-full px-4 py-3.5 border border-gray-200 rounded-xl shadow-sm placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent sm:text-sm transition duration-200 ease-in-out bg-gray-50 focus:bg-white">
                  </div>
                </div>

                <div class="space-y-1">
                  <label for="password" class="block text-sm font-medium text-gray-700">Password</label>
                  <div class="mt-1">
                    <input id="password" name="password" type="password" autocomplete="current-password" required [(ngModel)]="password"
                      class="appearance-none block w-full px-4 py-3.5 border border-gray-200 rounded-xl shadow-sm placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent sm:text-sm transition duration-200 ease-in-out bg-gray-50 focus:bg-white">
                  </div>
                </div>

                <div class="flex items-center justify-between">
                  <div class="flex items-center">
                    <input id="remember-me" name="remember-me" type="checkbox" [(ngModel)]="rememberMe" class="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 rounded transition duration-150 ease-in-out">
                    <label for="remember-me" class="ml-2 block text-sm text-gray-900">Remember me</label>
                  </div>

                  <div class="text-sm">
                    <a routerLink="/forgot-password" class="font-medium text-primary-600 hover:text-primary-500 transition-colors">Forgot your password?</a>
                  </div>
                </div>

                <!-- CAPTCHA -->
                <div class="mt-4">
                  <app-turnstile-captcha (tokenChange)="captchaToken = $event" />
                </div>

                <div>
                  <button type="submit" [disabled]="loading()"
                    class="w-full flex justify-center py-3.5 px-4 border border-transparent rounded-xl shadow-lg shadow-primary-500/30 text-sm font-bold text-white bg-gradient-to-r from-primary-600 to-primary-500 hover:from-primary-700 hover:to-primary-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500 disabled:opacity-50 disabled:cursor-not-allowed transition-all duration-200 transform hover:-translate-y-0.5">
                    @if (loading()) {
                      <span class="flex items-center">
                        <svg class="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                        Signing in...
                      </span>
                    } @else {
                      <span>Sign in</span>
                    }
                  </button>
                </div>
              </form>
            }

            <!-- TOTP Step -->
            @if (step() === 'totp') {
              <form class="space-y-6" (ngSubmit)="onVerify2FA()">
                <div>
                  <label for="totp" class="block text-sm font-medium text-gray-700">Authentication Code</label>
                  <div class="mt-1">
                    <input id="totp" name="totp" type="text" inputmode="numeric" pattern="[0-9]*" maxlength="6" required [(ngModel)]="totpCode"
                      placeholder="000000"
                      class="appearance-none block w-full px-4 py-3.5 border border-gray-200 rounded-xl shadow-sm placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent text-center text-2xl tracking-widest font-mono bg-gray-50 focus:bg-white">
                  </div>
                  <p class="mt-2 text-xs text-gray-500">Enter the code from Google Authenticator or your authenticator app</p>
                </div>

                <div class="flex gap-3">
                  <button type="button" (click)="cancelTotp()"
                    class="flex-1 py-3.5 px-4 border border-gray-300 rounded-xl shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500 transition-all duration-200">
                    Back
                  </button>
                  <button type="submit" [disabled]="loading() || totpCode.length !== 6"
                    class="flex-1 flex justify-center py-3.5 px-4 border border-transparent rounded-xl shadow-lg shadow-primary-500/30 text-sm font-bold text-white bg-gradient-to-r from-primary-600 to-primary-500 hover:from-primary-700 hover:to-primary-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500 disabled:opacity-50 disabled:cursor-not-allowed transition-all duration-200">
                    @if (loading()) {
                      <span class="flex items-center">
                        <svg class="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                        Verifying...
                      </span>
                    } @else {
                      <span>Verify</span>
                    }
                  </button>
                </div>
              </form>
            }

            @if (step() === 'credentials') {
              <div class="mt-8">
                <div class="relative">
                  <div class="absolute inset-0 flex items-center">
                    <div class="w-full border-t border-gray-200"></div>
                  </div>
                  <div class="relative flex justify-center text-sm">
                    <span class="px-4 bg-white text-gray-500 font-medium">Or continue with</span>
                  </div>
                </div>

                <div class="mt-8 grid grid-cols-1 gap-3">
                  <button type="button" (click)="signInWithPasskey()" [disabled]="loading()"
                    class="w-full inline-flex justify-center py-3.5 px-4 border border-gray-200 rounded-xl shadow-sm bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 hover:text-gray-900 transition-all duration-200 transform hover:-translate-y-0.5 disabled:opacity-50">
                     <span class="sr-only">Passkey</span>
                     <svg class="h-5 w-5 text-gray-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                       <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
                     </svg>
                     <span class="ml-2">Sign in with Passkey</span>
                  </button>
                </div>
              </div>
              
              <p class="mt-8 text-center text-sm text-gray-600">
                 Don't have an account? 
                 <a routerLink="/register" class="font-medium text-primary-600 hover:text-primary-500 transition-colors">Sign up for free</a>
              </p>
            }
          </div>
        </div>
      </div>

      <!-- Right Side - Image/Gradient -->
      <div class="hidden lg:block relative w-0 flex-1 overflow-hidden bg-gray-900">
        <div class="absolute inset-0 bg-gradient-to-br from-gray-900 via-gray-800 to-primary-900">
           <div class="absolute inset-0 bg-[url('https://images.unsplash.com/photo-1557683316-973673baf926?ixlib=rb-1.2.1&auto=format&fit=crop&w=1950&q=80')] opacity-10 mix-blend-overlay bg-cover bg-center"></div>
           <div class="absolute inset-0 bg-gradient-to-t from-gray-900/90 to-transparent"></div>
           
           <div class="absolute inset-0 flex items-center justify-center z-10">
              <div class="text-center px-12 max-w-2xl">
                 <div class="mb-8 inline-flex items-center justify-center h-20 w-20 rounded-2xl bg-white/10 backdrop-blur-lg border border-white/20 shadow-2xl">
                    <svg class="h-10 w-10 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                    </svg>
                 </div>
                 <h2 class="text-5xl font-bold text-white mb-6 tracking-tight leading-tight">Secure Identity Management</h2>
                 <p class="text-xl text-gray-300 leading-relaxed">Enterprise-grade authentication, simplified. Protect your users with Passkeys and modern security standards.</p>
              </div>
           </div>
           
           <!-- Decorative elements -->
           <div class="absolute top-0 right-0 -mr-20 -mt-20 w-[500px] h-[500px] rounded-full bg-primary-500 opacity-20 blur-[100px] animate-pulse"></div>
           <div class="absolute bottom-0 left-0 -ml-20 -mb-20 w-[500px] h-[500px] rounded-full bg-secondary-500 opacity-20 blur-[100px] animate-pulse" style="animation-delay: 2s;"></div>
        </div>
      </div>
    </div>
  `,
  styles: [`
    :host {
      display: block;
    }
  `]
})
export class LoginComponent implements OnInit {
  email = '';
  password = '';
  totpCode = '';
  captchaToken = '';
  rememberMe = false;
  theme: Theme | null = null;

  // Signals
  loading = signal(false);
  step = signal<LoginStep>('credentials');

  private authService = inject(AuthService);
  private brandingService = inject(BrandingService);
  private errorService = inject(ErrorHandlerService);
  private route = inject(ActivatedRoute);

  ngOnInit() {
    this.route.queryParams.subscribe(params => {
      const tenantId = params['tenant_id'];
      this.theme = this.brandingService.applyTheme(tenantId);
    });
  }

  onSubmit() {
    this.loading.set(true);

    this.authService.login(
      this.email,
      this.password,
      this.captchaToken,
      undefined, // totpCode
      this.rememberMe
    ).subscribe({
      next: (response) => {
        this.loading.set(false);
        if (response.requires_2fa) {
          this.step.set('totp');
        }
        // If not 2FA, redirect handled by service
      },
      error: () => {
        // Error handled globally by interceptor -> ErrorHandlerService
        this.loading.set(false);
      }
    });
  }

  onVerify2FA() {
    this.loading.set(true);

    this.authService.verify2FA(this.totpCode).subscribe({
      next: () => {
        this.loading.set(false);
        // Redirect handled by service
      },
      error: () => {
        // Error handled globally by interceptor -> ErrorHandlerService
        this.loading.set(false);
      }
    });
  }

  cancelTotp() {
    this.step.set('credentials');
    this.totpCode = '';
    this.authService.tempToken.set('');
    this.authService.requires2FA.set(false);
  }

  async signInWithPasskey() {
    this.loading.set(true);

    try {
      // 1. Begin passkey login
      const options = await this.authService.beginPasskeyLogin().toPromise();

      // 2. Create credential using WebAuthn API
      const credential = await navigator.credentials.get({
        publicKey: {
          ...options,
          challenge: this.base64ToBuffer(options.publicKey.challenge),
          allowCredentials: options.publicKey.allowCredentials?.map((c: any) => ({
            ...c,
            id: this.base64ToBuffer(c.id)
          }))
        }
      }) as PublicKeyCredential;

      if (!credential) {
        throw new Error('No credential returned');
      }

      // 3. Prepare response for server
      const response = credential.response as AuthenticatorAssertionResponse;
      const credentialData = {
        id: credential.id,
        rawId: this.bufferToBase64(credential.rawId),
        type: credential.type,
        response: {
          clientDataJSON: this.bufferToBase64(response.clientDataJSON),
          authenticatorData: this.bufferToBase64(response.authenticatorData),
          signature: this.bufferToBase64(response.signature),
          userHandle: response.userHandle ? this.bufferToBase64(response.userHandle) : null
        }
      };

      // 4. Finish login
      await this.authService.finishPasskeyLogin(credentialData).toPromise();
      this.loading.set(false);
    } catch (err: any) {
      this.loading.set(false);
      if (err.name === 'NotAllowedError') {
        this.errorService.showWarning('Passkey authentication was cancelled');
      } else {
        this.errorService.showError(err.message || 'Passkey authentication failed');
      }
    }
  }

  // WebAuthn helpers
  private base64ToBuffer(base64: string): ArrayBuffer {
    const padding = '='.repeat((4 - base64.length % 4) % 4);
    const b64 = (base64 + padding).replace(/-/g, '+').replace(/_/g, '/');
    const raw = atob(b64);
    const arr = new Uint8Array(raw.length);
    for (let i = 0; i < raw.length; i++) {
      arr[i] = raw.charCodeAt(i);
    }
    return arr.buffer;
  }

  private bufferToBase64(buffer: ArrayBuffer): string {
    const bytes = new Uint8Array(buffer);
    let binary = '';
    for (let i = 0; i < bytes.byteLength; i++) {
      binary += String.fromCharCode(bytes[i]);
    }
    return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
  }
}

