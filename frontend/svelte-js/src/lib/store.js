import { writable } from 'svelte/store';

// Retrieve initial state from localStorage if available
const storage = typeof window === 'undefined' ? null : window.localStorage;
const storedAccessToken = storage?.getItem('access_token') || null;
const storedRefreshToken = storage?.getItem('refresh_token') || null;
let storedUser = null;
try { storedUser = JSON.parse(storage?.getItem('user') || 'null'); } catch { storage?.removeItem('user'); }

export const auth = writable({
  isAuthenticated: !!storedAccessToken,
  accessToken: storedAccessToken,
  refreshToken: storedRefreshToken,
  user: storedUser ? JSON.parse(storedUser) : null,
});

// Helper to update store and localStorage
export const setAuth = (data) => {
  const accessToken = data.accessToken ?? storedAccessToken;
  const refreshToken = data.refreshToken ?? storedRefreshToken;
  const user = data.user ?? storedUser;
  if (accessToken) storage?.setItem('access_token', accessToken);
  if (refreshToken) storage?.setItem('refresh_token', refreshToken);
  if (user) storage?.setItem('user', JSON.stringify(user));

  auth.set({
    isAuthenticated: true,
    accessToken,
    refreshToken,
    user,
  });
};

// Helper to clear auth
export const clearAuth = () => {
  storage?.removeItem('access_token');
  storage?.removeItem('refresh_token');
  storage?.removeItem('user');
  
  auth.set({
    isAuthenticated: false,
    accessToken: null,
    refreshToken: null,
    user: null,
  });
};
