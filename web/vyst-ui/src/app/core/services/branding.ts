import { Injectable, inject } from '@angular/core';
import { DOCUMENT } from '@angular/common';

export interface Theme {
    primaryColor: string;
    logoUrl?: string;
}

@Injectable({
    providedIn: 'root',
})
export class BrandingService {
    private document = inject(DOCUMENT);

    applyTheme(tenantId: string | null): Theme {
        const theme: Theme = {
            primaryColor: '#4f46e5', // Default Indigo
        };

        if (tenantId === 'demo') {
            theme.primaryColor = '#ff5722'; // Orange
            theme.logoUrl = 'https://via.placeholder.com/150x50?text=Demo+Corp';
        }

        this.setPrimaryColor(theme.primaryColor);
        return theme;
    }

    private setPrimaryColor(color: string) {
        this.document.documentElement.style.setProperty('--primary-color', color);
    }
}
