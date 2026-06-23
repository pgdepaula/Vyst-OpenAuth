import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router } from '@angular/router';
import { tap, map, switchMap } from 'rxjs/operators';
import { Observable, of } from 'rxjs';
import { AppConfigService } from './app-config.service';
import { SessionService, StoredUserData } from './session.service';
import { Apollo, gql } from 'apollo-angular';

export interface LoginResponse {
  token?: string;
  refresh_token?: string;
  expires_in?: number;
  requires_2fa?: boolean;
  temp_token?: string;
  user?: {
    id: string;
    email: string;
    tenant_id?: string;
    roles?: string[];
  };
}

export interface CaptchaConfig {
  site_key: string;
  enabled: boolean;
}

export interface TOTPSetupResponse {
  secret: string;
  qr_code_url: string;
  backup_codes: string[];
}

@Injectable({
  providedIn: 'root',
})
export class AuthService {
  private http = inject(HttpClient);
  private router = inject(Router);
  private configService = inject(AppConfigService);
  private sessionService = inject(SessionService);
  private apollo = inject(Apollo);

  // Signals for state
  currentUser = signal<any>(null);
  requires2FA = signal<boolean>(false);
  tempToken = signal<string>('');

  // Remember me state for current login attempt
  private loginRememberMe = false;

  // CAPTCHA config (cached)
  private captchaConfig: CaptchaConfig | null = null;

  login(
    email: string,
    password: string,
    captchaToken?: string,
    totpCode?: string,
    rememberMe = false
  ): Observable<LoginResponse> {
    const url = `${this.configService.apiUrl}/auth/login`;
    const body: Record<string, unknown> = { email, password };
    if (captchaToken) body['captcha_token'] = captchaToken;
    if (totpCode) body['totp_code'] = totpCode;

    // Store remember me preference for use in setSession
    this.loginRememberMe = rememberMe;

    return this.http.post<LoginResponse>(url, body).pipe(
      tap((response) => {
        if (response.requires_2fa) {
          this.requires2FA.set(true);
          this.tempToken.set(response.temp_token ?? '');
        } else if (response.token) {
          this.requires2FA.set(false);
          this.tempToken.set('');
          this.setSession(response);
        }
      })
    );
  }

  // Uses GraphQL for fetching profile
  getProfile(): Observable<any> {
    const GET_ME = gql`
      query GetMe {
        me {
          id
          email
          tenant_id
          status
        }
      }
    `;

    return this.apollo.watchQuery<any>({
      query: GET_ME,
      fetchPolicy: 'network-only'
    }).valueChanges.pipe(
      map(result => result.data.me),
      tap(user => {
        this.currentUser.set({ ...this.currentUser(), ...user });
      })
    );
  }

  verify2FA(totpCode: string): Observable<LoginResponse> {
    const url = `${this.configService.apiUrl}/auth/login`;
    const body = {
      temp_token: this.tempToken(),
      totp_code: totpCode
    };

    return this.http.post<LoginResponse>(url, body).pipe(
      tap((response) => {
        if (response.token) {
          this.requires2FA.set(false);
          this.tempToken.set('');
          this.setSession(response);
        }
      })
    );
  }

  register(data: any, captchaToken?: string): Observable<any> {
    const url = `${this.configService.apiUrl}/auth/register`;
    const body = { ...data };
    if (captchaToken) body.captcha_token = captchaToken;
    return this.http.post<any>(url, body);
  }

  requestPasswordReset(email: string, captchaToken?: string): Observable<any> {
    const url = `${this.configService.apiUrl}/auth/forgot-password`;
    const body: Record<string, string> = { email };
    if (captchaToken) body['captcha_token'] = captchaToken;
    return this.http.post(url, body);
  }

  resetPassword(token: string, password: string): Observable<any> {
    const url = `${this.configService.apiUrl}/auth/reset-password`;
    return this.http.post(url, { token, password });
  }

  logout() {
    // Use SessionService for proper cleanup
    this.sessionService.clearSession(false);

    // Clear local state
    this.currentUser.set(null);
    this.requires2FA.set(false);
    this.tempToken.set('');

    // Clear Apollo cache
    this.apollo.client.resetStore();

    // Navigate to login
    this.router.navigate(['/login']);
  }

  private setSession(response: LoginResponse) {
    const token = response.token!;
    const expiresIn = response.expires_in ?? 3600; // Default 1 hour

    // Build user data from response or fetch later
    const userData: StoredUserData = response.user ?? {
      id: '',
      email: '',
    };

    // Save session with SessionService
    this.sessionService.saveSession({
      accessToken: token,
      refreshToken: response.refresh_token,
      expiresIn,
      user: userData,
      rememberMe: this.loginRememberMe,
    });

    // Update local state
    this.currentUser.set({ token, ...userData });

    // Fetch profile to get full user data
    this.getProfile().subscribe();

    // Navigate to company selection
    this.router.navigate(['/auth/company-selection']);
  }

  getToken(): string | null {
    return this.sessionService.accessToken();
  }

  isAuthenticated(): boolean {
    return this.sessionService.isAuthenticated();
  }

  /**
   * Initialize authentication state on app startup.
   * Should be called in APP_INITIALIZER.
   */
  async initializeAuth(): Promise<void> {
    await this.sessionService.initialize();

    if (this.sessionService.isAuthenticated()) {
      // Restore user from session
      const user = this.sessionService.user();
      if (user) {
        this.currentUser.set(user);
      }

      // Check if token needs refresh
      if (this.sessionService.needsRefresh()) {
        await this.refreshTokenSilently();
      }
    }
  }

  /**
   * Silently refresh the access token.
   */
  private async refreshTokenSilently(): Promise<void> {
    const refreshToken = this.sessionService.getRefreshToken();
    if (!refreshToken) return;

    try {
      const url = `${this.configService.apiUrl}/auth/refresh`;
      const response = await this.http
        .post<{ token: string; expires_in: number }>(url, {
          refresh_token: refreshToken,
        })
        .toPromise();

      if (response?.token) {
        this.sessionService.updateAccessToken(
          response.token,
          response.expires_in
        );
      }
    } catch {
      // Refresh failed, logout
      this.logout();
    }
  }

  // ============================================
  // CAPTCHA Methods
  // ============================================

  getCaptchaConfig(): Observable<CaptchaConfig> {
    if (this.captchaConfig) {
      return of(this.captchaConfig);
    }
    const url = `${this.configService.apiUrl}/auth/captcha-config`;
    return this.http.get<CaptchaConfig>(url).pipe(
      tap(config => this.captchaConfig = config)
    );
  }

  // ============================================
  // 2FA Methods
  // ============================================

  setup2FA(): Observable<TOTPSetupResponse> {
    const url = `${this.configService.apiUrl}/auth/2fa/setup`;
    return this.http.post<TOTPSetupResponse>(url, {});
  }

  enable2FA(code: string): Observable<any> {
    const url = `${this.configService.apiUrl}/auth/2fa/verify`;
    return this.http.post(url, { code });
  }

  get2FAStatus(): Observable<{ enabled: boolean }> {
    const url = `${this.configService.apiUrl}/auth/2fa/status`;
    return this.http.get<{ enabled: boolean }>(url);
  }

  disable2FA(code?: string): Observable<any> {
    let url = `${this.configService.apiUrl}/auth/2fa`;
    if (code) url += `?code=${code}`;
    return this.http.delete(url);
  }

  // ============================================
  // WebAuthn/Passkey Methods
  // ============================================

  beginPasskeyLogin(): Observable<any> {
    const url = `${this.configService.apiUrl}/auth/passkeys/login/begin`;
    return this.http.post(url, {});
  }

  finishPasskeyLogin(credential: unknown): Observable<LoginResponse> {
    const url = `${this.configService.apiUrl}/auth/passkeys/login/finish`;
    return this.http.post<LoginResponse>(url, credential).pipe(
      tap((response) => {
        if (response.token) {
          this.setSession(response);
        }
      })
    );
  }

  beginPasskeyRegistration(): Observable<any> {
    const url = `${this.configService.apiUrl}/auth/passkeys/register/begin`;
    return this.http.post(url, {});
  }

  finishPasskeyRegistration(credential: any): Observable<any> {
    const url = `${this.configService.apiUrl}/auth/passkeys/register/finish`;
    return this.http.post(url, credential);
  }
  // ============================================
  // Company Methods
  // ============================================

  switchCompany(companyId: string): Observable<LoginResponse> {
    const url = `${this.configService.apiUrl}/api/v1/auth/switch-company`;
    return this.http.post<LoginResponse>(url, { company_id: companyId }).pipe(
      tap((response) => {
        if (response.token) {
          // Update session with new token and potentially new user context
          this.setSession(response);
          this.router.navigate(['/admin/dashboard']); // Redirect to dashboard after switch
        }
      })
    );
  }
}

