
import { Injectable, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';

export interface AppError {
    message: string;
    code?: string; // e.g. "AUTH_INVALID_CREDENTIALS" or "401"
    status?: number;
    type: 'error' | 'warning' | 'info';
    timestamp: number;
}

@Injectable({
    providedIn: 'root'
})
export class ErrorHandlerService {
    // Signal to hold the current active error (or null)
    activeError = signal<AppError | null>(null);

    // Auto-dismiss timer
    private timeoutId: any;

    showError(message: string, code?: string, status?: number) {
        this.setError({
            message,
            code,
            status,
            type: 'error',
            timestamp: Date.now()
        });
    }

    showWarning(message: string) {
        this.setError({
            message,
            type: 'warning',
            timestamp: Date.now()
        });
    }

    showInfo(message: string) {
        this.setError({
            message,
            type: 'info',
            timestamp: Date.now()
        });
    }

    handleHttpError(error: HttpErrorResponse) {
        let message = 'An unexpected error occurred';
        let code = error.status.toString();

        // 1. Try to parse "error" field from REST API (standard Vyst API format)
        if (error.error && typeof error.error === 'object') {
            if (error.error.error) {
                message = error.error.error; // e.g. { "error": "Invalid credentials" }
            } else if (error.error.message) {
                message = error.error.message;
            }
        } else if (typeof error.error === 'string') {
            // Sometimes backend might send plain string
            message = error.error;
        }

        // 2. Map standard status codes if no specific message
        if (message === 'An unexpected error occurred') {
            switch (error.status) {
                case 400: message = 'Invalid request.'; break;
                case 401: message = 'Authentication failed. Please login again.'; break;
                case 403: message = 'You do not have permission to perform this action.'; break;
                case 404: message = 'Resource not found.'; break;
                case 429: message = 'Too many requests. Please try again later.'; break;
                case 500: message = 'Server error. Please try again later.'; break;
                case 503: message = 'Service unavailable.'; break;
                case 0: message = 'Network error. Please check your connection.'; break;
            }
        }

        this.showError(message, code, error.status);
        console.error('API Error:', error);
    }

    handleGraphQLError(errors: any[]) {
        if (!errors || errors.length === 0) return;

        // Just show the first error for now
        const first = errors[0];
        const message = first.message || 'GraphQL Error';
        const code = first.extensions?.code || 'GQL_ERR';

        this.showError(message, code);
    }

    clear() {
        this.activeError.set(null);
        if (this.timeoutId) clearTimeout(this.timeoutId);
    }

    private setError(err: AppError) {
        this.activeError.set(err);

        // Auto clear after 5 seconds
        if (this.timeoutId) clearTimeout(this.timeoutId);
        this.timeoutId = setTimeout(() => {
            this.activeError.set(null);
        }, 6000);
    }
}
