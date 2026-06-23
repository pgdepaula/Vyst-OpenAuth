// K6 Helper Library - Shared functions for all API tests
// =============================================================================

import { RefinedResponse, ResponseType } from 'k6/http';

// Configuration from environment variables
export const CONFIG = {
    REST_URL: __ENV.REST_URL || 'http://localhost:8080',
    GRPC_URL: __ENV.GRPC_URL || 'localhost:52151',
    GRAPHQL_URL: __ENV.GRAPHQL_URL || 'http://localhost:8080/api/v1/graphql',
};

// Test user interface
export interface TestUser {
    email: string;
    password: string;
    tenant_name: string;
}

// Auth state interface
export interface AuthState {
    accessToken: string | null;
    refreshToken: string | null;
    userId: string | null;
    tenantId: string | null;
}

// Generate unique test data
export function generateTestUser(): TestUser {
    const timestamp = Date.now();
    const random = Math.random().toString(36).substring(7);
    return {
        email: `k6_test_${timestamp}_${random}@example.com`,
        password: 'K6TestPass123!',
        tenant_name: `K6_Tenant_${timestamp}`,
    };
}

// Standard HTTP headers
export function jsonHeaders(token: string | null = null): Record<string, string> {
    const headers: Record<string, string> = {
        'Content-Type': 'application/json',
    };
    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }
    return headers;
}

// Parse JSON response safely
export function parseJson<T>(res: RefinedResponse<ResponseType>): T | null {
    try {
        return JSON.parse(res.body as string) as T;
    } catch {
        return null;
    }
}

// GraphQL request body builder
export function buildGraphQLBody(query: string, variables: Record<string, unknown> = {}): string {
    return JSON.stringify({
        query,
        variables,
    });
}

// Sleep duration randomizer (for rate limiting)
export function randomSleep(min: number = 0.1, max: number = 0.3): number {
    return Math.random() * (max - min) + min;
}
