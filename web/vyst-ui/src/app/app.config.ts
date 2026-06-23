
import { ApplicationConfig, provideZoneChangeDetection, APP_INITIALIZER, inject } from '@angular/core';
import { provideRouter } from '@angular/router';
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { provideApollo } from 'apollo-angular';
import { HttpLink } from 'apollo-angular/http';
import { InMemoryCache } from '@apollo/client/core';

import { routes } from './app.routes';
import { AppConfigService } from './core/services/app-config.service';
import { errorInterceptor } from './core/error.interceptor';

export function initializeApp(appConfig: AppConfigService) {
  return () => appConfig.loadConfig();
}

export const appConfig: ApplicationConfig = {
  providers: [
    provideZoneChangeDetection({ eventCoalescing: true }),
    provideRouter(routes),
    provideHttpClient(
      withInterceptors([errorInterceptor])
    ),
    provideApollo(() => {
      const httpLink = inject(HttpLink);
      const configService = inject(AppConfigService);
      const apiUrl = configService.apiUrl || '';
      // fallback empty if config not loaded yet, though APP_INITIALIZER should run first.
      // For Apollo, we might need a workaround if config isn't ready.
      // Assuming config is loaded before Apollo query is made.

      return {
        link: httpLink.create({ uri: `${apiUrl}/query` }),
        cache: new InMemoryCache(),
      };
    }),
    {
      provide: APP_INITIALIZER,
      useFactory: initializeApp,
      deps: [AppConfigService],
      multi: true
    }
  ]
};
