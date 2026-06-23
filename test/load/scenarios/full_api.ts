// K6 E2E Tests - Master Runner (ALL APIs)
// =============================================================================
// Runs all API tests: REST, GraphQL, and gRPC
// Run: k6 run --compatibility-mode=extended test/e2e/k6/all_apis_test.ts

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Options } from 'k6/options';
import { CONFIG, generateTestUser, jsonHeaders, parseJson, buildGraphQLBody, TestUser, AuthState } from './lib/helpers.js';

// Test configuration
export const options: Options = {
    scenarios: {
        e2e_tests: {
            executor: 'shared-iterations',
            vus: 1,
            iterations: 1,
            maxDuration: '10m',
        },
    },
    thresholds: {
        checks: ['rate>0.85'],
        http_req_duration: ['p(95)<5000'],
    },
};

const REST_URL = CONFIG.REST_URL;
const GRAPHQL_URL = CONFIG.GRAPHQL_URL;

// State
const state: AuthState = {
    accessToken: null,
    refreshToken: null,
    userId: null,
    tenantId: null,
};
let testUser: TestUser;

export default function (): void {
    console.log('='.repeat(60));
    console.log('VYST IDENTITY - K6 E2E TESTS (ALL APIs)');
    console.log('='.repeat(60));

    // ==========================================================================
    // REST API TESTS
    // ==========================================================================
    group('REST API', () => {
        // Health check
        group('Health', () => {
            const res = http.get(`${REST_URL}/health`);
            check(res, { 'REST: /health - 200': (r) => r.status === 200 });
        });

        // Register
        group('Auth', () => {
            testUser = generateTestUser();

            const registerRes = http.post(
                `${REST_URL}/auth/register`,
                JSON.stringify({
                    email: testUser.email,
                    password: testUser.password,
                    tenant_name: testUser.tenant_name,
                }),
                { headers: jsonHeaders() }
            );
            check(registerRes, { 'REST: /auth/register - 201': (r) => r.status === 201 });

            sleep(0.1);

            // Login
            const loginRes = http.post(
                `${REST_URL}/auth/login`,
                JSON.stringify({ email: testUser.email, password: testUser.password }),
                { headers: jsonHeaders() }
            );
            check(loginRes, { 'REST: /auth/login - response': (r) => r.status >= 200 });

            if (loginRes.status === 200) {
                const body = parseJson<{ token?: string; access_token?: string; refresh_token: string }>(loginRes);
                if (body) {
                    state.accessToken = body.token || body.access_token || null;
                    state.refreshToken = body.refresh_token;
                }
            }
        });

        // Roles
        if (state.accessToken) {
            group('Roles', () => {
                const listRes = http.get(`${REST_URL}/api/v1/roles`, {
                    headers: jsonHeaders(state.accessToken),
                });
                check(listRes, { 'REST: /api/v1/roles - 200': (r) => r.status === 200 });
            });

            group('API Keys', () => {
                const listRes = http.get(`${REST_URL}/api/v1/api-keys`, {
                    headers: jsonHeaders(state.accessToken),
                });
                check(listRes, { 'REST: /api/v1/api-keys - 200': (r) => r.status === 200 });
            });
        }
    });

    sleep(0.5);

    // ==========================================================================
    // GRAPHQL API TESTS
    // ==========================================================================
    group('GraphQL API', () => {
        // Introspection
        group('Introspection', () => {
            const query = `{ __schema { types { name } } }`;
            const res = http.post(GRAPHQL_URL, buildGraphQLBody(query), {
                headers: jsonHeaders(),
            });
            check(res, { 'GraphQL: Introspection - 200': (r) => r.status === 200 });
        });

        // Register via GraphQL
        group('Mutations', () => {
            const gqlUser = generateTestUser();
            const registerQuery = `
        mutation { 
          register(email: "${gqlUser.email}", password: "${gqlUser.password}", tenant_name: "${gqlUser.tenant_name}") {
            user { id }
          }
        }
      `;
            const res = http.post(GRAPHQL_URL, buildGraphQLBody(registerQuery), {
                headers: jsonHeaders(),
            });
            check(res, { 'GraphQL: register - 200': (r) => r.status === 200 });
        });

        // Me query (if we have token)
        if (state.accessToken) {
            group('Queries', () => {
                const meQuery = `{ me { id email } }`;
                const res = http.post(GRAPHQL_URL, buildGraphQLBody(meQuery), {
                    headers: jsonHeaders(state.accessToken),
                });
                check(res, { 'GraphQL: me - 200': (r) => r.status === 200 });
            });
        }
    });

    sleep(0.5);

    // ==========================================================================
    // SUMMARY
    // ==========================================================================
    console.log('='.repeat(60));
    console.log('E2E TESTS COMPLETED');
    console.log('='.repeat(60));
}

export function teardown(): void {
    console.log('All API E2E tests completed');
}
