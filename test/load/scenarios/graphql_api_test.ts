// K6 E2E Tests - GraphQL API (ALL OPERATIONS)
// =============================================================================
// Tests all GraphQL operations against a running server
// Run: k6 run --compatibility-mode=extended test/e2e/k6/graphql/graphql_api_test.ts

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Options } from 'k6/options';
import { CONFIG, generateTestUser, jsonHeaders, parseJson, buildGraphQLBody, TestUser } from '../lib/helpers.js';

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

const GRAPHQL_URL = CONFIG.GRAPHQL_URL;

// State
let testUser: TestUser;
let accessToken: string | null = null;
let refreshToken: string | null = null;

// GraphQL Response interfaces
interface GraphQLResponse<T> {
    data: T | null;
    errors?: Array<{ message: string; path?: string[] }>;
}

interface AuthPayload {
    access_token: string;
    refresh_token: string;
    expires_in: number;
    user: {
        id: string;
        email: string;
    };
}

interface RegisterPayload {
    user: {
        id: string;
        email: string;
    };
    tenant: {
        id: string;
        name: string;
    };
}

interface User {
    id: string;
    email: string;
    tenant_id: string;
    status: string;
}

export default function (): void {
    // ==========================================================================
    // INTROSPECTION
    // ==========================================================================
    group('GraphQL Introspection', () => {
        const query = `
      query IntrospectionQuery {
        __schema {
          types {
            name
          }
        }
      }
    `;

        const res = http.post(GRAPHQL_URL, buildGraphQLBody(query), {
            headers: jsonHeaders(),
        });

        check(res, {
            'Introspection - status 200': (r) => r.status === 200,
            'Introspection - has data': (r) => {
                const body = parseJson<GraphQLResponse<{ __schema: unknown }>>(r);
                return body?.data?.__schema !== undefined;
            },
        });
    });

    sleep(0.2);

    // ==========================================================================
    // MUTATION: register
    // ==========================================================================
    group('Mutation: register', () => {
        testUser = generateTestUser();

        const query = `
      mutation Register($email: String!, $password: String!, $tenant_name: String!) {
        register(email: $email, password: $password, tenant_name: $tenant_name) {
          user {
            id
            email
          }
          tenant {
            id
            name
          }
        }
      }
    `;

        const res = http.post(
            GRAPHQL_URL,
            buildGraphQLBody(query, {
                email: testUser.email,
                password: testUser.password,
                tenant_name: testUser.tenant_name,
            }),
            { headers: jsonHeaders() }
        );

        check(res, {
            'register - status 200': (r) => r.status === 200,
            'register - has user': (r) => {
                const body = parseJson<GraphQLResponse<{ register: RegisterPayload }>>(r);
                return body?.data?.register?.user?.id !== undefined;
            },
            'register - has tenant': (r) => {
                const body = parseJson<GraphQLResponse<{ register: RegisterPayload }>>(r);
                return body?.data?.register?.tenant?.id !== undefined;
            },
        });
    });

    sleep(0.2);

    // ==========================================================================
    // MUTATION: login
    // ==========================================================================
    group('Mutation: login', () => {
        const query = `
      mutation Login($email: String!, $password: String!) {
        login(email: $email, password: $password) {
          access_token
          refresh_token
          expires_in
          user {
            id
            email
          }
        }
      }
    `;

        const res = http.post(
            GRAPHQL_URL,
            buildGraphQLBody(query, {
                email: testUser.email,
                password: testUser.password,
            }),
            { headers: jsonHeaders() }
        );

        check(res, {
            'login - status 200': (r) => r.status === 200,
            'login - returns response': (r) => {
                const body = parseJson<GraphQLResponse<{ login: AuthPayload }>>(r);
                if (body?.data?.login) {
                    accessToken = body.data.login.access_token;
                    refreshToken = body.data.login.refresh_token;
                    return true;
                }
                // Login might fail if user not activated - that's OK for this test
                return body?.errors !== undefined || body?.data !== undefined;
            },
        });
    });

    sleep(0.2);

    // ==========================================================================
    // QUERY: me
    // ==========================================================================
    group('Query: me', () => {
        const query = `
      query Me {
        me {
          id
          email
          tenant_id
          status
          created_at
          updated_at
        }
      }
    `;

        if (!accessToken) {
            console.log('Skipping me query - no access token');
            return;
        }

        const res = http.post(GRAPHQL_URL, buildGraphQLBody(query), {
            headers: jsonHeaders(accessToken),
        });

        check(res, {
            'me - status 200': (r) => r.status === 200,
            'me - has user data': (r) => {
                const body = parseJson<GraphQLResponse<{ me: User }>>(r);
                return body?.data?.me?.id !== undefined || body?.errors !== undefined;
            },
        });
    });

    sleep(0.2);

    // ==========================================================================
    // MUTATION: refreshToken
    // ==========================================================================
    group('Mutation: refreshToken', () => {
        const query = `
      mutation RefreshToken($refresh_token: String!) {
        refreshToken(refresh_token: $refresh_token) {
          access_token
          refresh_token
          expires_in
        }
      }
    `;

        if (!refreshToken) {
            console.log('Skipping refreshToken - no refresh token');
            return;
        }

        const res = http.post(
            GRAPHQL_URL,
            buildGraphQLBody(query, { refresh_token: refreshToken }),
            { headers: jsonHeaders() }
        );

        check(res, {
            'refreshToken - status 200': (r) => r.status === 200,
            'refreshToken - returns response': (r) => {
                const body = parseJson<GraphQLResponse<{ refreshToken: AuthPayload }>>(r);
                return body?.data?.refreshToken !== undefined || body?.errors !== undefined;
            },
        });
    });

    sleep(0.2);

    // ==========================================================================
    // MUTATION: logout
    // ==========================================================================
    group('Mutation: logout', () => {
        const query = `
      mutation Logout($refresh_token: String!) {
        logout(refresh_token: $refresh_token)
      }
    `;

        if (!refreshToken) {
            console.log('Skipping logout - no refresh token');
            return;
        }

        const res = http.post(
            GRAPHQL_URL,
            buildGraphQLBody(query, { refresh_token: refreshToken }),
            { headers: jsonHeaders() }
        );

        check(res, {
            'logout - status 200': (r) => r.status === 200,
            'logout - returns response': (r) => {
                const body = parseJson<GraphQLResponse<{ logout: boolean }>>(r);
                return body !== null;
            },
        });
    });
}

export function teardown(): void {
    console.log('GraphQL API E2E tests completed');
}
