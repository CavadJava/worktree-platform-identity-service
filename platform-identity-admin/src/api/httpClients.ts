import { createApiClient } from './httpClient';

export const platformIdentityApi = createApiClient(import.meta.env.VITE_PLATFORM_IDENTITY_API_URL);
