
import { HttpInterceptorFn, HttpErrorResponse } from '@angular/common/http';
import { inject } from '@angular/core';
import { catchError } from 'rxjs/operators';
import { throwError } from 'rxjs';
import { ErrorHandlerService } from './services/error-handler.service';
import { Router } from '@angular/router';

export const errorInterceptor: HttpInterceptorFn = (req, next) => {
    const errorService = inject(ErrorHandlerService);
    const router = inject(Router);

    return next(req).pipe(
        catchError((error: HttpErrorResponse) => {
            // Skip if it's a 404 (handled by components mostly) or if request explicitly wants to handle errors
            // For this "Global" requirement, we default to showing errors unless suppressed.
            // We can check a custom context token if needed later.

            if (error.status === 401) {
                // Optionally redirect to login, but let auth service handle state
                // router.navigate(['/login']);
            }

            errorService.handleHttpError(error);
            return throwError(() => error);
        })
    );
};
