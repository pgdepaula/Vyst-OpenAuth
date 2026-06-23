
import { TestBed } from '@angular/core/testing';
import { ErrorHandlerService } from './error-handler.service';
import { HttpErrorResponse } from '@angular/common/http';

describe('ErrorHandlerService', () => {
    let service: ErrorHandlerService;

    beforeEach(() => {
        TestBed.configureTestingModule({});
        service = TestBed.inject(ErrorHandlerService);
    });

    it('should be created', () => {
        expect(service).toBeTruthy();
    });

    it('should parse REST error object correctly', () => {
        const errorResponse = new HttpErrorResponse({
            error: { error: 'Invalid credentials' },
            status: 401,
            statusText: 'Unauthorized'
        });

        service.handleHttpError(errorResponse);
        const active = service.activeError();

        expect(active).toBeTruthy();
        expect(active?.message).toBe('Invalid credentials');
        expect(active?.status).toBe(401);
    });

    it('should parse REST message field correctly', () => {
        const errorResponse = new HttpErrorResponse({
            error: { message: 'Something went wrong' },
            status: 500
        });

        service.handleHttpError(errorResponse);
        expect(service.activeError()?.message).toBe('Something went wrong');
    });

    it('should fallback to default status messages', () => {
        const errorResponse = new HttpErrorResponse({
            error: null, // No body
            status: 404
        });

        service.handleHttpError(errorResponse);
        expect(service.activeError()?.message).toBe('Resource not found.');
    });
});
