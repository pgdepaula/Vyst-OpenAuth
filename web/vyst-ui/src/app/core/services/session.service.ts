import { Injectable, inject, signal, computed } from '@angular/core';
import { Router } from '@angular/router';

/**
 * Session storage keys for secure token management.
 */
const STORAGE_KEYS = {
  ACCESS_TOKEN: 'vyst_access_token',
  REFRESH_TOKEN: 'vyst_refresh_token',
  REMEMBER_ME: 'vyst_remember_me',
  USER_DATA: 'vyst_user_data',
  TOKEN_EXPIRY: 'vyst_token_expiry',
  LAST_ACTIVITY: 'vyst_last_activity',
} as const;

/**
 * Configuration for session management.
 */
const SESSION_CONFIG = {
  /** Inactivity timeout in milliseconds (30 minutes) */
  INACTIVITY_TIMEOUT_MS: 30 * 60 * 1000,
  /** Token refresh threshold in milliseconds (5 minutes before expiry) */
  REFRESH_THRESHOLD_MS: 5 * 60 * 1000,
  /** Maximum session duration with remember me (30 days) */
  MAX_REMEMBER_DURATION_MS: 30 * 24 * 60 * 60 * 1000,
  /** Activity check interval in milliseconds (1 minute) */
  ACTIVITY_CHECK_INTERVAL_MS: 60 * 1000,
} as const;

/**
 * Stored user data interface.
 */
export interface StoredUserData {
  id: string;
  email: string;
  tenantId?: string;
  roles?: string[];
}

/**
 * Session state interface.
 */
export interface SessionState {
  isAuthenticated: boolean;
  user: StoredUserData | null;
  rememberMe: boolean;
  expiresAt: number | null;
}

/**
 * SessionService manages user authentication state persistence.
 *
 * Features:
 * - Secure token storage (localStorage for remember me, sessionStorage otherwise)
 * - Automatic token refresh before expiry
 * - Inactivity-based session timeout
 * - Cross-tab session synchronization
 * - Secure cleanup on logout
 *
 * @example
 * ```typescript
 * // In login component
 * sessionService.saveSession({
 *   accessToken: response.token,
 *   refreshToken: response.refresh_token,
 *   expiresIn: response.expires_in,
 *   user: { id: '...', email: '...' },
 *   rememberMe: true
 * });
 *
 * // In app initializer
 * if (sessionService.hasValidSession()) {
 *   await sessionService.restoreSession();
 * }
 * ```
 */
@Injectable({
  providedIn: 'root',
})
export class SessionService {
  private router = inject(Router);

  // Reactive state
  private readonly _accessToken = signal<string | null>(null);
  private readonly _refreshToken = signal<string | null>(null);
  private readonly _user = signal<StoredUserData | null>(null);
  private readonly _rememberMe = signal(false);
  private readonly _expiresAt = signal<number | null>(null);
  private readonly _isInitialized = signal(false);

  // Computed state
  readonly isAuthenticated = computed(() => !!this._accessToken());
  readonly user = computed(() => this._user());
  readonly accessToken = computed(() => this._accessToken());
  readonly sessionState = computed<SessionState>(() => ({
    isAuthenticated: this.isAuthenticated(),
    user: this._user(),
    rememberMe: this._rememberMe(),
    expiresAt: this._expiresAt(),
  }));

  // Activity tracking
  private activityCheckInterval: ReturnType<typeof setInterval> | null = null;
  private storageEventListener: ((e: StorageEvent) => void) | null = null;

  /**
   * Initialize the session service.
   * Call this in APP_INITIALIZER.
   */
  async initialize(): Promise<void> {
    if (this._isInitialized()) return;

    // Try to restore session
    this.restoreFromStorage();

    // Setup activity tracking
    this.setupActivityTracking();

    // Setup cross-tab synchronization
    this.setupStorageListener();

    this._isInitialized.set(true);
  }

  /**
   * Save a new session after successful login.
   */
  saveSession(params: {
    accessToken: string;
    refreshToken?: string;
    expiresIn: number; // seconds
    user: StoredUserData;
    rememberMe: boolean;
  }): void {
    const { accessToken, refreshToken, expiresIn, user, rememberMe } = params;
    const expiresAt = Date.now() + expiresIn * 1000;

    // Update signals
    this._accessToken.set(accessToken);
    this._refreshToken.set(refreshToken ?? null);
    this._user.set(user);
    this._rememberMe.set(rememberMe);
    this._expiresAt.set(expiresAt);

    // Persist to storage
    const storage = this.getStorage(rememberMe);

    storage.setItem(STORAGE_KEYS.ACCESS_TOKEN, accessToken);
    if (refreshToken) {
      storage.setItem(STORAGE_KEYS.REFRESH_TOKEN, refreshToken);
    }
    storage.setItem(STORAGE_KEYS.USER_DATA, JSON.stringify(user));
    storage.setItem(STORAGE_KEYS.TOKEN_EXPIRY, expiresAt.toString());
    storage.setItem(STORAGE_KEYS.REMEMBER_ME, rememberMe.toString());
    storage.setItem(STORAGE_KEYS.LAST_ACTIVITY, Date.now().toString());

    // If remember me changed, clean up old storage
    if (rememberMe) {
      this.clearStorage(sessionStorage);
    } else {
      this.clearStorage(localStorage);
    }

    this.updateLastActivity();
  }

  /**
   * Update access token after refresh.
   */
  updateAccessToken(accessToken: string, expiresIn: number): void {
    const expiresAt = Date.now() + expiresIn * 1000;

    this._accessToken.set(accessToken);
    this._expiresAt.set(expiresAt);

    const storage = this.getStorage(this._rememberMe());
    storage.setItem(STORAGE_KEYS.ACCESS_TOKEN, accessToken);
    storage.setItem(STORAGE_KEYS.TOKEN_EXPIRY, expiresAt.toString());
  }

  /**
   * Check if there's a potentially valid session to restore.
   */
  hasValidSession(): boolean {
    // Check both storages
    const localRemember = localStorage.getItem(STORAGE_KEYS.REMEMBER_ME);
    const sessionRemember = sessionStorage.getItem(STORAGE_KEYS.REMEMBER_ME);

    if (localRemember === 'true') {
      const token = localStorage.getItem(STORAGE_KEYS.ACCESS_TOKEN);
      const expiry = localStorage.getItem(STORAGE_KEYS.TOKEN_EXPIRY);
      if (token && expiry) {
        const expiresAt = parseInt(expiry, 10);
        // Valid if not expired or has refresh token
        if (expiresAt > Date.now() || localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN)) {
          return true;
        }
      }
    }

    if (sessionRemember === 'false' || sessionRemember === null) {
      const token = sessionStorage.getItem(STORAGE_KEYS.ACCESS_TOKEN);
      if (token) {
        return true;
      }
    }

    return false;
  }

  /**
   * Check if token needs refresh.
   */
  needsRefresh(): boolean {
    const expiresAt = this._expiresAt();
    if (!expiresAt) return false;

    const timeUntilExpiry = expiresAt - Date.now();
    return timeUntilExpiry < SESSION_CONFIG.REFRESH_THRESHOLD_MS;
  }

  /**
   * Get refresh token for token refresh.
   */
  getRefreshToken(): string | null {
    return this._refreshToken();
  }

  /**
   * Clear session and logout.
   */
  clearSession(redirectToLogin = true): void {
    // Clear signals
    this._accessToken.set(null);
    this._refreshToken.set(null);
    this._user.set(null);
    this._rememberMe.set(false);
    this._expiresAt.set(null);

    // Clear both storages
    this.clearStorage(localStorage);
    this.clearStorage(sessionStorage);

    // Stop activity tracking
    this.stopActivityTracking();

    // Redirect
    if (redirectToLogin) {
      this.router.navigate(['/login']);
    }
  }

  /**
   * Update last activity timestamp.
   */
  updateLastActivity(): void {
    const storage = this.getStorage(this._rememberMe());
    storage.setItem(STORAGE_KEYS.LAST_ACTIVITY, Date.now().toString());
  }

  /**
   * Check if session has timed out due to inactivity.
   */
  private isSessionTimedOut(): boolean {
    // Skip timeout check if remember me is enabled
    if (this._rememberMe()) {
      // Check max session duration for remember me
      const rememberExpiry = Date.now() - SESSION_CONFIG.MAX_REMEMBER_DURATION_MS;
      const storage = this.getStorage(true);
      const lastActivity = storage.getItem(STORAGE_KEYS.LAST_ACTIVITY);
      if (lastActivity && parseInt(lastActivity, 10) < rememberExpiry) {
        return true;
      }
      return false;
    }

    // Check inactivity timeout for regular sessions
    const storage = this.getStorage(false);
    const lastActivity = storage.getItem(STORAGE_KEYS.LAST_ACTIVITY);
    if (!lastActivity) return true;

    const inactiveFor = Date.now() - parseInt(lastActivity, 10);
    return inactiveFor > SESSION_CONFIG.INACTIVITY_TIMEOUT_MS;
  }

  /**
   * Restore session from storage.
   */
  private restoreFromStorage(): void {
    // Check localStorage first (remember me)
    let storage: Storage = localStorage;
    let rememberMe = localStorage.getItem(STORAGE_KEYS.REMEMBER_ME) === 'true';

    if (!rememberMe) {
      storage = sessionStorage;
    }

    const accessToken = storage.getItem(STORAGE_KEYS.ACCESS_TOKEN);
    const refreshToken = storage.getItem(STORAGE_KEYS.REFRESH_TOKEN);
    const userData = storage.getItem(STORAGE_KEYS.USER_DATA);
    const tokenExpiry = storage.getItem(STORAGE_KEYS.TOKEN_EXPIRY);

    if (!accessToken) return;

    // Check timeout
    if (this.isSessionTimedOut()) {
      this.clearSession(false);
      return;
    }

    // Restore state
    this._accessToken.set(accessToken);
    this._refreshToken.set(refreshToken);
    this._rememberMe.set(rememberMe);
    this._expiresAt.set(tokenExpiry ? parseInt(tokenExpiry, 10) : null);

    if (userData) {
      try {
        this._user.set(JSON.parse(userData));
      } catch {
        // Invalid JSON, clear session
        this.clearSession(false);
      }
    }
  }

  /**
   * Get appropriate storage based on remember me setting.
   */
  private getStorage(rememberMe: boolean): Storage {
    return rememberMe ? localStorage : sessionStorage;
  }

  /**
   * Clear all session keys from a storage.
   */
  private clearStorage(storage: Storage): void {
    Object.values(STORAGE_KEYS).forEach((key) => {
      storage.removeItem(key);
    });
  }

  /**
   * Setup activity tracking for timeout.
   */
  private setupActivityTracking(): void {
    if (typeof window === 'undefined') return;

    // Track user activity
    const activityEvents = ['mousedown', 'mousemove', 'keydown', 'scroll', 'touchstart'];
    const updateActivity = () => this.updateLastActivity();

    activityEvents.forEach((event) => {
      window.addEventListener(event, updateActivity, { passive: true });
    });

    // Periodic check for timeout
    this.activityCheckInterval = setInterval(() => {
      if (this.isAuthenticated() && this.isSessionTimedOut()) {
        this.clearSession(true);
      }
    }, SESSION_CONFIG.ACTIVITY_CHECK_INTERVAL_MS);
  }

  /**
   * Stop activity tracking.
   */
  private stopActivityTracking(): void {
    if (this.activityCheckInterval) {
      clearInterval(this.activityCheckInterval);
      this.activityCheckInterval = null;
    }
  }

  /**
   * Setup cross-tab session synchronization.
   */
  private setupStorageListener(): void {
    if (typeof window === 'undefined') return;

    this.storageEventListener = (event: StorageEvent) => {
      // Handle logout in other tab
      if (event.key === STORAGE_KEYS.ACCESS_TOKEN && event.newValue === null) {
        this.clearSession(true);
      }

      // Handle login in other tab
      if (event.key === STORAGE_KEYS.ACCESS_TOKEN && event.newValue && !this._accessToken()) {
        this.restoreFromStorage();
      }
    };

    window.addEventListener('storage', this.storageEventListener);
  }

  /**
   * Cleanup on service destroy.
   */
  ngOnDestroy(): void {
    this.stopActivityTracking();
    if (this.storageEventListener && typeof window !== 'undefined') {
      window.removeEventListener('storage', this.storageEventListener);
    }
  }
}
