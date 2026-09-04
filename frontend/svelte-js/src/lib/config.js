const configuredAPI = import.meta.env.API_URL || import.meta.env.VITE_API_URL || 'http://localhost:8080/auth';

export const API_URL = configuredAPI.replace(/\/+$/, '');
export const GATEWAY_URL = API_URL.endsWith('/auth') ? API_URL.slice(0, -5) : API_URL;
export const SITE_HOST = import.meta.env.SITE_HOST || import.meta.env.VITE_SITE_HOST || window.location.hostname;
