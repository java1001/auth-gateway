import { writable } from 'svelte/store';

// Retrieve initial state from localStorage if available
const storedAccessToken = localStorage.getItem('access_token');
const storedRefreshToken = localStorage.getItem('refresh_token');
const storedUser = localStorage.getItem('user');

export const auth = writable({
  isAuthenticated: !!storedAccessToken,
  accessToken: storedAccessToken,
  refreshToken: storedRefreshToken,
  user: storedUser ? JSON.parse(storedUser) : null,
});

// Helper to update store and localStorage
export const setAuth = (data) => {
  if (data.accessToken) localStorage.setItem('access_token', data.accessToken);
  if (data.refreshToken) localStorage.setItem('refresh_token', data.refreshToken);
  if (data.user) localStorage.setItem('user', JSON.stringify(data.user));

  auth.set({
    isAuthenticated: true,
    accessToken: data.accessToken,
    refreshToken: data.refreshToken,
    user: data.user,
  });
};

// Helper to clear auth
export const clearAuth = () => {
  localStorage.removeItem('access_token');
  localStorage.removeItem('refresh_token');
  localStorage.removeItem('user');
  
  auth.set({
    isAuthenticated: false,
    accessToken: null,
    refreshToken: null,
    user: null,
  });
};
