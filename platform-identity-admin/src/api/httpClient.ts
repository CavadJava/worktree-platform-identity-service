import axios, { type AxiosInstance } from 'axios';
import type { ApiError } from './types';

interface Envelope<T> {
  success: boolean;
  data?: T;
  error?: ApiError;
}

export class ApiRequestError extends Error {
  code: string;
  status: number;

  constructor(status: number, error: ApiError) {
    super(error.message);
    this.name = 'ApiRequestError';
    this.code = error.code;
    this.status = status;
  }
}

export function unwrapEnvelope<T>(status: number, body: Envelope<T>): T {
  if (body.success) {
    return body.data as T;
  }
  throw new ApiRequestError(status, body.error ?? { code: 'error', message: 'Unknown error' });
}

export function createApiClient(baseURL: string): AxiosInstance {
  const client = axios.create({ baseURL });

  client.interceptors.request.use((config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.set('Authorization', `Bearer ${token}`);
    }
    return config;
  });

  client.interceptors.response.use(
    (response) => {
      if (response.status === 204 || !response.data) {
        return response;
      }
      response.data = unwrapEnvelope(response.status, response.data);
      return response;
    },
    (error) => {
      if (error.response) {
        if (error.response.status === 401) {
          const hadToken = !!localStorage.getItem('token');
          if (hadToken) {
            localStorage.removeItem('token');
            window.location.href = '/login';
          }
        }
        if (error.response.data) {
          try {
            unwrapEnvelope(error.response.status, error.response.data);
          } catch (unwrapped) {
            return Promise.reject(unwrapped);
          }
        }
      }
      return Promise.reject(error);
    }
  );

  return client;
}
