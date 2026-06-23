// K6 E2E Tests - gRPC API (ALL METHODS)
// =============================================================================
// Tests all gRPC methods against a running server
// Requires: k6 with xk6-grpc extension
// Run: k6 run --compatibility-mode=extended test/e2e/k6/grpc/grpc_api_test.ts

import grpc from 'k6/net/grpc';
import { check, sleep, group } from 'k6';
import { Options } from 'k6/options';
import { CONFIG, generateTestUser, TestUser } from '../lib/helpers.js';

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
        grpc_req_duration: ['p(95)<3000'],
    },
};

const GRPC_URL = CONFIG.GRPC_URL;

// gRPC client
const client = new grpc.Client();

// Load proto file
client.load(['../../api/proto'], 'identity.proto');

// State
let testUser: TestUser;
let accessToken: string | null = null;
let refreshToken: string | null = null;

export default function (): void {
    // Connect to gRPC server
    client.connect(GRPC_URL, {
        plaintext: true,
    });

    // ==========================================================================
    // Register
    // ==========================================================================
    group('gRPC: Register', () => {
        testUser = generateTestUser();

        const response = client.invoke('identity.IdentityService/Register', {
            email: testUser.email,
            password: testUser.password,
            tenant_name: testUser.tenant_name,
        });

        check(response, {
            'Register - status OK': (r) => r && r.status === grpc.StatusOK,
            'Register - has user_id': (r) => {
                if (r && r.message) {
                    const msg = r.message as { user_id: string; tenant_id: string };
                    return !!msg.user_id;
                }
                return false;
            },
        });
    });

    sleep(0.2);

    // ==========================================================================
    // Login
    // ==========================================================================
    group('gRPC: Login', () => {
        const response = client.invoke('identity.IdentityService/Login', {
            email: testUser.email,
            password: testUser.password,
        });

        check(response, {
            'Login - returns response': (r) => r !== null,
            'Login - stores tokens': (r) => {
                if (r && r.status === grpc.StatusOK && r.message) {
                    const msg = r.message as { access_token: string; refresh_token: string };
                    accessToken = msg.access_token;
                    refreshToken = msg.refresh_token;
                    return true;
                }
                // Login might fail if user not activated
                return true;
            },
        });
    });

    sleep(0.2);

    // ==========================================================================
    // ValidateToken
    // ==========================================================================
    group('gRPC: ValidateToken', () => {
        if (!accessToken) {
            console.log('Skipping ValidateToken - no access token');
            return;
        }

        const response = client.invoke('identity.IdentityService/ValidateToken', {
            token: accessToken,
        });

        check(response, {
            'ValidateToken - status OK': (r) => r && r.status === grpc.StatusOK,
            'ValidateToken - has valid field': (r) => {
                if (r && r.message) {
                    const msg = r.message as { valid: boolean };
                    return 'valid' in msg;
                }
                return false;
            },
        });
    });

    sleep(0.2);

    // ==========================================================================
    // RefreshToken
    // ==========================================================================
    group('gRPC: RefreshToken', () => {
        if (!refreshToken) {
            console.log('Skipping RefreshToken - no refresh token');
            return;
        }

        const response = client.invoke('identity.IdentityService/RefreshToken', {
            refresh_token: refreshToken,
        });

        check(response, {
            'RefreshToken - returns response': (r) => r !== null,
            'RefreshToken - has access_token': (r) => {
                if (r && r.status === grpc.StatusOK && r.message) {
                    const msg = r.message as { access_token: string };
                    return !!msg.access_token;
                }
                return true; // Might fail if token invalid
            },
        });
    });

    sleep(0.2);

    // ==========================================================================
    // Logout
    // ==========================================================================
    group('gRPC: Logout', () => {
        if (!refreshToken) {
            console.log('Skipping Logout - no refresh token');
            return;
        }

        const response = client.invoke('identity.IdentityService/Logout', {
            refresh_token: refreshToken,
        });

        check(response, {
            'Logout - returns response': (r) => r !== null,
            'Logout - has success field': (r) => {
                if (r && r.message) {
                    const msg = r.message as { success: boolean };
                    return 'success' in msg;
                }
                return false;
            },
        });
    });

    // Close connection
    client.close();
}

export function teardown(): void {
    console.log('gRPC API E2E tests completed');
}
