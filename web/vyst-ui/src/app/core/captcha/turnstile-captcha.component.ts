import {
  Component,
  OnInit,
  OnDestroy,
  inject,
  output,
  signal,
  ChangeDetectionStrategy,
} from '@angular/core';
import { CaptchaService } from './captcha.service';

/**
 * Cloudflare Turnstile CAPTCHA component.
 *
 * This component handles the complete CAPTCHA lifecycle:
 * - Fetches configuration from the backend
 * - Loads the Turnstile script
 * - Renders the widget
 * - Emits tokens on successful verification
 * - Handles errors and retries
 *
 * @example
 * ```html
 * <app-turnstile-captcha (tokenChange)="captchaToken = $event" />
 * ```
 *
 * The component is fully accessible and follows WCAG AA guidelines.
 */
@Component({
  selector: 'app-turnstile-captcha',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (loading()) {
      <div
        class="flex items-center justify-center py-4"
        role="status"
        aria-label="Loading CAPTCHA"
      >
        <div
          class="animate-spin h-6 w-6 border-2 border-primary-500 border-t-transparent rounded-full"
        ></div>
        <span class="sr-only">Loading CAPTCHA...</span>
      </div>
    }

    @if (error()) {
      <div class="text-red-600 text-sm py-2" role="alert">
        {{ error() }}
        <button
          type="button"
          (click)="retry()"
          class="ml-2 underline hover:no-underline focus:outline-none focus:ring-2 focus:ring-primary-500 rounded px-1"
          aria-label="Retry loading CAPTCHA"
        >
          Retry
        </button>
      </div>
    }

    <div
      [id]="containerId"
      [class.hidden]="loading() || error() || !enabled()"
      role="group"
      aria-label="CAPTCHA verification"
    ></div>

    @if (!enabled() && !loading()) {
      <span class="sr-only">CAPTCHA verification is not required</span>
    }
  `,
  styles: [
    `
      :host {
        display: block;
        min-height: 65px;
      }

      .hidden {
        display: none;
      }
    `,
  ],
})
export class TurnstileCaptchaComponent implements OnInit, OnDestroy {
  private readonly captchaService = inject(CaptchaService);

  /** Emits the CAPTCHA token when verification succeeds, or empty string on failure */
  readonly tokenChange = output<string>();

  /** Loading state signal */
  readonly loading = signal(true);

  /** Error message signal */
  readonly error = signal<string | null>(null);

  /** Whether CAPTCHA is enabled */
  readonly enabled = signal(false);

  /** Unique container ID for this widget */
  readonly containerId = `turnstile-${Math.random().toString(36).substring(2, 11)}`;

  /** Widget ID returned by Turnstile */
  private widgetId: string | undefined;

  async ngOnInit(): Promise<void> {
    await this.initializeCaptcha();
  }

  ngOnDestroy(): void {
    if (this.widgetId) {
      this.captchaService.remove(this.widgetId);
    }
  }

  /**
   * Initialize the CAPTCHA widget.
   */
  private async initializeCaptcha(): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      // Get configuration from backend
      const config = await new Promise<{ enabled: boolean; site_key: string }>(
        (resolve, reject) => {
          this.captchaService.getConfig().subscribe({
            next: resolve,
            error: reject,
          });
        }
      );

      this.enabled.set(config.enabled);

      if (!config.enabled) {
        this.loading.set(false);
        // Emit empty token to signal CAPTCHA is not required
        this.tokenChange.emit('');
        return;
      }

      // Load Turnstile script
      await this.captchaService.loadScript();

      // Small delay to ensure DOM is ready
      await new Promise((resolve) => setTimeout(resolve, 100));

      // Render widget
      this.widgetId = this.captchaService.render(
        this.containerId,
        (token: string) => {
          this.tokenChange.emit(token);
        }
      );

      if (!this.widgetId) {
        throw new Error('Failed to render CAPTCHA widget');
      }

      this.loading.set(false);
    } catch (err) {
      this.loading.set(false);
      this.error.set('Failed to load CAPTCHA. Please refresh the page.');
      console.error('CAPTCHA initialization error:', err);
    }
  }

  /**
   * Retry loading the CAPTCHA.
   */
  async retry(): Promise<void> {
    // Clear any existing widget
    if (this.widgetId) {
      this.captchaService.remove(this.widgetId);
      this.widgetId = undefined;
    }

    // Clear cache to force fresh config
    this.captchaService.clearCache();

    await this.initializeCaptcha();
  }

  /**
   * Reset the CAPTCHA widget for re-verification.
   * Call this after a form submission failure.
   */
  reset(): void {
    if (this.widgetId) {
      this.captchaService.reset(this.widgetId);
    }
  }
}
