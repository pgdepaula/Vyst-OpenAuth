import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { tap } from 'rxjs/operators';
import { firstValueFrom } from 'rxjs';

export interface AppConfig {
    apiUrl: string;
}

@Injectable({
    providedIn: 'root'
})
export class AppConfigService {
    private http = inject(HttpClient);

    // Signal to hold the configuration
    config = signal<AppConfig>({ apiUrl: '' });

    loadConfig(): Promise<void> {
        return firstValueFrom(
            this.http.get<AppConfig>('/config.json').pipe(
                tap(config => {
                    console.log('Configuration loaded:', config);
                    this.config.set(config);
                })
            )
        ).then(() => undefined);
    }

    get apiUrl(): string {
        return this.config().apiUrl;
    }
}
