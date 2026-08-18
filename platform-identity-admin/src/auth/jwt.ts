import type { JwtClaims } from '../api/types';

export function decodeToken(token: string): JwtClaims {
  const payloadSegment = token.split('.')[1];
  if (!payloadSegment) {
    throw new Error('malformed token: missing payload segment');
  }
  const base64 = payloadSegment.replace(/-/g, '+').replace(/_/g, '/');
  const json = atob(base64);
  return JSON.parse(json) as JwtClaims;
}

export function isExpired(claims: JwtClaims): boolean {
  return claims.exp * 1000 <= Date.now();
}
