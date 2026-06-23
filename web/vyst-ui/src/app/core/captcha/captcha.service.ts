import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, of, tap, catchError } from 'rxjs';
import { AppConfigService } from '../services/app-config.service';

/**
 * Configuration returned from the backend for CAPTCHA.
 */
export interface CaptchaConfig {
  site_key: string;
  enabled: boolean;
}

/**
 * Global Turnstile interface for TypeScript.
 */
declare global {
  interface Window {
    turnstile?: {
      render: (
        container: string | HTMLElement,
        options: TurnstileRenderOptions
      ) => string;
      reset: (widgetId: string) => void;
      remove: (widgetId: string) => void;
    };
  }
}

interface TurnstileRenderOptions {
  sitekey: string;
  callback: (token: string) => void;
  'error-callback'?: () => void;
  'expired-callback'?: () => void;
  theme?: 'light' | 'dark' | 'auto';
  size?: 'normal' | 'compact';
  tabindex?: number;
  action?: string;
}

/**
 * Service for managing CAPTCHA (Cloudflare Turnstile) integration.
 *
 * This service handles:
 * - Loading the Turnstile script dynamically
 * - Fetching CAPTCHA configuration from the backend
 * - Rendering and managing CAPTCHA widgets
 *
 * @example
 * ```typescript
 * class MyComponent {
 *   captchaService = inject(CaptchaService);
 *
 *   async ngOnInit() {
 *     const config = await firstValueFrom(this.captchaService.getConfig());
 *     if (config.enabled) {
 *       await this.captchaService.loadScript();
 *     }
 *   }
 * }
 * ```
 */
@Injectable({
  providedIn: 'root',
})
export class CaptchaService {
  private readonly http = inject(HttpClient);
  private readonly configService = inject(AppConfigService);

  /** Cached CAPTCHA configuration from backend */
  private cachedConfig: CaptchaConfig | null = null;

  /** Signal indicating whether CAPTCHA is enabled */
  readonly isEnabled = signal(false);

  /** Signal containing the CAPTCHA site key */
  readonly siteKey = signal('');

  /** Signal indicating whether the Turnstile script is loaded */
  readonly isLoaded = signal(false);

  /**
   * Fetches CAPTCHA configuration from the backend.
   * Results are cached after the first call.
   *
   * @returns Observable with the CAPTCHA configuration
   */
  getConfig(): Observable<CaptchaConfig> {
    if (this.cachedConfig) {
      return of(this.cachedConfig);
    }

    const url = `${this.configService.apiUrl}/auth/captcha-config`;
    return this.http.get<CaptchaConfig>(url).pipe(
      tap((config) => {
        this.cachedConfig = config;
        this.isEnabled.set(config.enabled);
        this.siteKey.set(config.site_key);
      }),
      catchError(() => {
        // If config fetch fails, assume CAPTCHA is disabled
        const defaultConfig: CaptchaConfig = { site_key: '', enabled: false };
        this.cachedConfig = defaultConfig;
        this.isEnabled.set(false);
        return of(defaultConfig);
      })
    );
  }

  /**
   * Loads the Cloudflare Turnstile script if not already loaded.
   * Uses explicit rendering mode for better control.
   *
   * @returns Promise that resolves when the script is loaded
   */
  loadScript(): Promise<void> {
    return new Promise((resolve, reject) => {
      // Already loaded
      if (this.isLoaded()) {
        resolve();
        return;
      }

      // Check if script already exists in DOM
      if (document.querySelector('script[src*="turnstile"]')) {
        this.isLoaded.set(true);
        resolve();
        return;
      }

      const script = document.createElement('script');
      script.src =
        'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit';
      script.async = true;
      script.defer = true;

      script.onload = () => {
        this.isLoaded.set(true);
        resolve();
      };

      script.onerror = () => {
        reject(new Error('Failed to load Cloudflare Turnstile script'));
      };

      document.head.appendChild(script);
    });
  }

  /**
   * Renders a Turnstile widget in the specified container.
   *
   * @param containerId - ID of the HTML element to render the widget in
   * @param callback - Function called with the token when verification succeeds
   * @param options - Optional configuration for the widget
   * @returns Widget ID for later reference, or undefined if rendering failed
   */
  render(
    containerId: string,
    callback: (token: string) => void,
    options?: Partial<TurnstileRenderOptions>
  ): string | undefined {
    if (!window.turnstile) {
      console.error('Turnstile not loaded');
      return undefined;
    }

    const siteKey = this.siteKey();
    if (!siteKey) {
      console.error('CAPTCHA site key not configured');
      return undefined;
    }

    try {
      return window.turnstile.render(`#${containerId}`, {
        sitekey: siteKey,
        callback,
        'error-callback': () => callback(''),
        'expired-callback': () => callback(''),
        theme: 'light',
        ...options,
      });
    } catch (error) {
      console.error('Failed to render Turnstile widget:', error);
      return undefined;
    }
  }

  /**
   * Resets a Turnstile widget to allow re-verification.
   *
   * @param widgetId - The widget ID returned from render()
   */
  reset(widgetId: string): void {
    if (window.turnstile && widgetId) {
      try {
        window.turnstile.reset(widgetId);
      } catch (error) {
        console.error('Failed to reset Turnstile widget:', error);
      }
    }
  }

  /**
   * Removes a Turnstile widget from the DOM.
   *
   * @param widgetId - The widget ID returned from render()
   */
  remove(widgetId: string): void {
    if (window.turnstile && widgetId) {
      try {
        window.turnstile.remove(widgetId);
      } catch (error) {
        console.error('Failed to remove Turnstile widget:', error);
      }
    }
  }

  /**
   * Clears the cached configuration, forcing a fresh fetch on next getConfig() call.
   */
  clearCache(): void {
    this.cachedConfig = null;
  }
}
