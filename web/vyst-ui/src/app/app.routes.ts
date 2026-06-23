import { Routes } from '@angular/router';
import { authGuard } from './core/guards/auth.guard';

export const routes: Routes = [
    // Auth routes - lazy loaded
    {
        path: 'login',
        loadComponent: () => import('./features/auth/login/login').then(m => m.LoginComponent)
    },
    {
        path: 'register',
        loadComponent: () => import('./features/auth/register/register').then(m => m.RegisterComponent)
    },
    {
        path: 'forgot-password',
        loadComponent: () => import('./features/auth/forgot-password/forgot-password').then(m => m.ForgotPasswordComponent)
    },
    {
        path: 'reset-password',
        loadComponent: () => import('./features/auth/reset-password/reset-password').then(m => m.ResetPasswordComponent)
    },
    {
        path: 'auth/verify-email',
        loadComponent: () => import('./features/auth/verify-email/verify-email').then(m => m.VerifyEmailComponent)
    },
    {
        path: 'auth/company-selection',
        loadComponent: () => import('./features/auth/company-login/company-selection.component').then(m => m.CompanySelectionComponent),
        canActivate: [authGuard]
    },

    // Onboarding
    {
        path: 'onboarding',
        loadComponent: () => import('./features/onboarding/setup-organization/setup-organization').then(m => m.SetupOrganizationComponent),
        canActivate: [authGuard]
    },

    // Admin routes
    {
        path: 'admin',
        loadComponent: () => import('./features/admin/layout/layout').then(m => m.LayoutComponent),
        canActivate: [authGuard],
        children: [
            {
                path: 'tenants',
                loadComponent: () => import('./features/admin/super-admin/tenants-list/tenants-list').then(m => m.TenantsListComponent)
            },
            {
                path: 'roles',
                loadComponent: () => import('./features/admin/access-management/roles-list/roles-list').then(m => m.RolesListComponent)
            },
            {
                path: 'roles/:id',
                loadComponent: () => import('./features/admin/access-management/role-editor/role-editor').then(m => m.RoleEditorComponent)
            },
            {
                path: 'api-keys',
                loadComponent: () => import('./features/admin/api-keys/api-keys').then(m => m.ApiKeysComponent)
            },
            { path: '', redirectTo: 'tenants', pathMatch: 'full' }
        ]
    },

    // Default redirect
    { path: '', redirectTo: 'login', pathMatch: 'full' },
];
