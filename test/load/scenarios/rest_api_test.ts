// K6 E2E Tests - REST API (ALL ENDPOINTS)
// =============================================================================
// Tests all REST API endpoints against a running server
// Run: k6 run --compatibility-mode=extended test/e2e/k6/rest/rest_api_test.ts

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Options } from 'k6/options';
import { CONFIG, generateTestUser, jsonHeaders, parseJson, TestUser, AuthState } from '../lib/helpers.js';

// Test configuration
export const options: Options = {
    scenarios: {
        e2e_tests: {
            executor: 'shared-iterations',
            vus: 1,
            iterations: 1,
            maxDuration: '5m',
        },
    },
    thresholds: {
        checks: ['rate>0.90'],
        http_req_duration: ['p(95)<3000'],
    },
};

const BASE_URL = CONFIG.REST_URL;

// Store state across test groups
const state: AuthState = {
    accessToken: null,
    refreshToken: null,
    userId: null,
    tenantId: null,
};

let testUser: TestUser;
let roleId: string | null = null;
let apiKeyId: string | null = null;

export default function (): void {
    // ==========================================================================
    // HEALTH ENDPOINTS
    // ==========================================================================
    group('Health Endpoints', () => {
        const healthRes = http.get(`${BASE_URL}/health`);
        check(healthRes, {
            'GET /health - status 200': (r) => r.status === 200,
        });

        const readyRes = http.get(`${BASE_URL}/ready`);
        check(readyRes, {
            'GET /ready - status 200': (r) => r.status === 200,
        });
    });

    sleep(0.2);

    // ==========================================================================
    // AUTH ENDPOINTS
    // ==========================================================================
    group('Auth Endpoints', () => {
        testUser = generateTestUser();

        // POST /auth/register
        const registerRes = http.post(
            `${BASE_URL}/auth/register`,
            JSON.stringify({
                email: testUser.email,
                password: testUser.password,
                tenant_name: testUser.tenant_name,
            }),
            { headers: jsonHeaders() }
        );
        check(registerRes, {
            'POST /auth/register - status 201': (r) => r.status === 201,
            'POST /auth/register - has user_id': (r) => {
                const body = parseJson<{ user_id: string; tenant_id: string }>(r);
                if (body) {
                    state.userId = body.user_id;
                    state.tenantId = body.tenant_id;
                    return !!state.userId;
                }
                return false;
            },
        });

        sleep(0.1);

        // POST /auth/login
        const loginRes = http.post(
            `${BASE_URL}/auth/login`,
            JSON.stringify({
                email: testUser.email,
                password: testUser.password,
            }),
            { headers: jsonHeaders() }
        );
        check(loginRes, {
            'POST /auth/login - returns response': (r) => r.status >= 200 && r.status < 500,
        });

        if (loginRes.status === 200) {
            const body = parseJson<{ token?: string; access_token?: string; refresh_token: string }>(loginRes);
            if (body) {
                state.accessToken = body.token || body.access_token || null;
                state.refreshToken = body.refresh_token;
            }
        }

        sleep(0.1);

        // GET /auth/captcha-config
        const captchaRes = http.get(`${BASE_URL}/auth/captcha-config`);
        check(captchaRes, {
            'GET /auth/captcha-config - status 200': (r) => r.status === 200,
            'GET /auth/captcha-config - has enabled field': (r) => {
                const body = parseJson<{ enabled: boolean }>(r);
                return body !== null && 'enabled' in body;
            },
        });

        // GET /auth/verify-email (test with invalid token)
        const verifyRes = http.get(`${BASE_URL}/auth/verify-email?token=invalid-test-token`);
        check(verifyRes, {
            'GET /auth/verify-email - handles invalid token': (r) => r.status === 400 || r.status === 401,
        });

        // GET /auth/me (requires auth)
        if (state.accessToken) {
            const meRes = http.get(`${BASE_URL}/auth/me`, {
                headers: jsonHeaders(state.accessToken),
            });
            check(meRes, {
                'GET /auth/me - status 200': (r) => r.status === 200,
                'GET /auth/me - has email': (r) => {
                    const body = parseJson<{ email: string }>(r);
                    return body?.email === testUser.email;
                },
            });
        }

        // POST /auth/refresh
        if (state.refreshToken) {
            const refreshRes = http.post(
                `${BASE_URL}/auth/refresh`,
                JSON.stringify({ refresh_token: state.refreshToken }),
                { headers: jsonHeaders() }
            );
            check(refreshRes, {
                'POST /auth/refresh - status 200': (r) => r.status === 200,
                'POST /auth/refresh - has new token': (r) => {
                    const body = parseJson<{ token?: string; access_token?: string }>(r);
                    return !!(body?.token || body?.access_token);
                },
            });
        }
    });

    sleep(0.2);

    // ==========================================================================
    // PASSWORD ENDPOINTS
    // ==========================================================================
    group('Password Endpoints', () => {
        const forgotRes = http.post(
            `${BASE_URL}/auth/forgot-password`,
            JSON.stringify({ email: testUser.email }),
            { headers: jsonHeaders() }
        );
        check(forgotRes, {
            'POST /auth/forgot-password - status 200': (r) => r.status === 200,
        });

        sleep(0.1);

        const resetRes = http.post(
            `${BASE_URL}/auth/reset-password`,
            JSON.stringify({
                token: 'invalid-reset-token',
                new_password: 'NewPass123!',
            }),
            { headers: jsonHeaders() }
        );
        check(resetRes, {
            'POST /auth/reset-password - handles invalid token': (r) => r.status === 400,
        });
    });

    sleep(0.2);

    // ==========================================================================
    // ROLES ENDPOINTS
    // ==========================================================================
    group('Roles Endpoints', () => {
        if (!state.accessToken) {
            console.log('Skipping roles tests - no access token');
            return;
        }

        // POST /api/v1/roles
        const createRes = http.post(
            `${BASE_URL}/api/v1/roles`,
            JSON.stringify({
                name: 'K6 Test Role',
                description: 'Created by K6 E2E tests',
                permissions: ['read:users', 'write:users'],
            }),
            { headers: jsonHeaders(state.accessToken) }
        );
        check(createRes, {
            'POST /api/v1/roles - status 201': (r) => r.status === 201,
            'POST /api/v1/roles - has id': (r) => {
                if (r.status === 201) {
                    const body = parseJson<{ id: string }>(r);
                    if (body) {
                        roleId = body.id;
                        return !!roleId;
                    }
                }
                return false;
            },
        });

        sleep(0.1);

        // GET /api/v1/roles
        const listRes = http.get(`${BASE_URL}/api/v1/roles`, {
            headers: jsonHeaders(state.accessToken),
        });
        check(listRes, {
            'GET /api/v1/roles - status 200': (r) => r.status === 200,
            'GET /api/v1/roles - is array': (r) => {
                const body = parseJson<unknown[]>(r);
                return Array.isArray(body);
            },
        });

        if (roleId) {
            // GET /api/v1/roles/{id}
            const getRes = http.get(`${BASE_URL}/api/v1/roles/${roleId}`, {
                headers: jsonHeaders(state.accessToken),
            });
            check(getRes, {
                'GET /api/v1/roles/{id} - status 200': (r) => r.status === 200,
            });

            sleep(0.1);

            // PUT /api/v1/roles/{id}
            const updateRes = http.put(
                `${BASE_URL}/api/v1/roles/${roleId}`,
                JSON.stringify({
                    name: 'K6 Test Role Updated',
                    description: 'Updated by K6',
                    permissions: ['read:users'],
                }),
                { headers: jsonHeaders(state.accessToken) }
            );
            check(updateRes, {
                'PUT /api/v1/roles/{id} - status 200': (r) => r.status === 200,
            });

            sleep(0.1);

            // DELETE /api/v1/roles/{id}
            const deleteRes = http.del(`${BASE_URL}/api/v1/roles/${roleId}`, null, {
                headers: jsonHeaders(state.accessToken),
            });
            check(deleteRes, {
                'DELETE /api/v1/roles/{id} - status 204 or 200': (r) => r.status === 204 || r.status === 200,
            });
        }
    });

    sleep(0.2);

    // ==========================================================================
    // TENANTS ENDPOINTS
    // ==========================================================================
    group('Tenants Endpoints', () => {
        if (!state.accessToken) return;

        const createRes = http.post(
            `${BASE_URL}/api/v1/tenants`,
            JSON.stringify({ name: 'K6 New Tenant' }),
            { headers: jsonHeaders(state.accessToken) }
        );
        check(createRes, {
            'POST /api/v1/tenants - returns response': (r) => r.status >= 200 && r.status < 500,
        });

        const listRes = http.get(`${BASE_URL}/api/v1/admin/tenants`, {
            headers: jsonHeaders(state.accessToken),
        });
        check(listRes, {
            'GET /api/v1/admin/tenants - returns response': (r) =>
                r.status === 200 || r.status === 403 || r.status === 401,
        });

        const suspendRes = http.post(
            `${BASE_URL}/api/v1/admin/tenants/test-id/suspend`,
            null,
            { headers: jsonHeaders(state.accessToken) }
        );
        check(suspendRes, {
            'POST /api/v1/admin/tenants/{id}/suspend - returns response': (r) =>
                r.status >= 200 && r.status < 500,
        });
    });

    sleep(0.2);

    // ==========================================================================
    // API KEYS ENDPOINTS
    // ==========================================================================
    group('API Keys Endpoints', () => {
        if (!state.accessToken) return;

        const createRes = http.post(
            `${BASE_URL}/api/v1/api-keys`,
            JSON.stringify({
                name: 'K6 Test API Key',
                expires_in_days: 30,
            }),
            { headers: jsonHeaders(state.accessToken) }
        );
        check(createRes, {
            'POST /api/v1/api-keys - status 201': (r) => r.status === 201,
            'POST /api/v1/api-keys - has key': (r) => {
                if (r.status === 201) {
                    const body = parseJson<{ id: string; key: string }>(r);
                    if (body) {
                        apiKeyId = body.id;
                        return !!body.key;
                    }
                }
                return false;
            },
        });

        sleep(0.1);

        const listRes = http.get(`${BASE_URL}/api/v1/api-keys`, {
            headers: jsonHeaders(state.accessToken),
        });
        check(listRes, {
            'GET /api/v1/api-keys - status 200': (r) => r.status === 200,
        });

        if (apiKeyId) {
            const deleteRes = http.del(`${BASE_URL}/api/v1/api-keys/${apiKeyId}`, null, {
                headers: jsonHeaders(state.accessToken),
            });
            check(deleteRes, {
                'DELETE /api/v1/api-keys/{id} - status 204 or 200': (r) =>
                    r.status === 204 || r.status === 200,
            });
        }
    });

    sleep(0.2);

    // ==========================================================================
    // 2FA/TOTP ENDPOINTS
    // ==========================================================================
    group('2FA/TOTP Endpoints', () => {
        if (!state.accessToken) return;

        const statusRes = http.get(`${BASE_URL}/auth/2fa/status`, {
            headers: jsonHeaders(state.accessToken),
        });
        check(statusRes, {
            'GET /auth/2fa/status - returns response': (r) => r.status >= 200 && r.status < 500,
        });

        const setupRes = http.post(`${BASE_URL}/auth/2fa/setup`, null, {
            headers: jsonHeaders(state.accessToken),
        });
        check(setupRes, {
            'POST /auth/2fa/setup - returns response': (r) => r.status >= 200 && r.status < 500,
        });

        const verifyRes = http.post(
            `${BASE_URL}/auth/2fa/verify`,
            JSON.stringify({ code: '000000' }),
            { headers: jsonHeaders(state.accessToken) }
        );
        check(verifyRes, {
            'POST /auth/2fa/verify - handles invalid code': (r) => r.status >= 200 && r.status < 500,
        });

        const disableRes = http.del(`${BASE_URL}/auth/2fa`, null, {
            headers: jsonHeaders(state.accessToken),
        });
        check(disableRes, {
            'DELETE /auth/2fa - returns response': (r) => r.status >= 200 && r.status < 500,
        });
    });

    sleep(0.2);

    // ==========================================================================
    // WEBAUTHN ENDPOINTS
    // ==========================================================================
    group('WebAuthn Endpoints', () => {
        if (!state.accessToken) return;

        const registerBeginRes = http.post(`${BASE_URL}/auth/webauthn/register/begin`, null, {
            headers: jsonHeaders(state.accessToken),
        });
        check(registerBeginRes, {
            'POST /auth/webauthn/register/begin - returns response': (r) =>
                r.status >= 200 && r.status < 500,
        });

        const loginBeginRes = http.post(`${BASE_URL}/auth/webauthn/login/begin`, null, {
            headers: jsonHeaders(state.accessToken),
        });
        check(loginBeginRes, {
            'POST /auth/webauthn/login/begin - returns response': (r) =>
                r.status >= 200 && r.status < 500,
        });
    });

    sleep(0.2);

    // ==========================================================================
    // STATS ENDPOINT
    // ==========================================================================
    group('Stats Endpoint', () => {
        if (!state.accessToken) return;

        const statsRes = http.get(`${BASE_URL}/api/v1/stats`, {
            headers: jsonHeaders(state.accessToken),
        });
        check(statsRes, {
            'GET /api/v1/stats - returns response': (r) => r.status >= 200 && r.status < 500,
        });
    });

    sleep(0.2);

    // ==========================================================================
    // CLEANUP
    // ==========================================================================
    group('Cleanup', () => {
        if (state.refreshToken) {
            const logoutRes = http.post(
                `${BASE_URL}/auth/logout`,
                JSON.stringify({ refresh_token: state.refreshToken }),
                { headers: jsonHeaders() }
            );
            check(logoutRes, {
                'POST /auth/logout - status 200': (r) => r.status === 200,
            });
        }
    });
}

export function teardown(): void {
    console.log('REST API E2E tests completed');
}
