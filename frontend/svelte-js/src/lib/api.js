import axios from 'axios';
import { auth, setAuth, clearAuth } from './store.js';
import { get } from 'svelte/store';
import { navigate } from 'svelte-routing';

// Fallback to localhost if not defined
const API_URL = import.meta.env.API_URL || 'http://localhost:8080/auth'; 

const api = axios.create({
  baseURL: API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Intercept requests to inject the access token
api.interceptors.request.use(
  (config) => {
    const state = get(auth);
    if (state.accessToken) {
      config.headers.Authorization = `Bearer ${state.accessToken}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Intercept responses to handle 401 Unauthorized (Token Expiry)
let isRefreshing = false;
let failedQueue = [];

const processQueue = (error, token = null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error);
    } else {
      prom.resolve(token);
    }
  });
  failedQueue = [];
};

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;

    if (error.response && error.response.status === 401 && !originalRequest._retry) {
      if (isRefreshing) {
        return new Promise(function (resolve, reject) {
          failedQueue.push({ resolve, reject });
        })
          .then((token) => {
            originalRequest.headers.Authorization = 'Bearer ' + token;
            return api(originalRequest);
          })
          .catch((err) => {
            return Promise.reject(err);
          });
      }

      originalRequest._retry = true;
      isRefreshing = true;

      const state = get(auth);
      const refreshToken = state.refreshToken;

      if (!refreshToken) {
        clearAuth();
        navigate('/login', { replace: true });
        return Promise.reject(error);
      }

      try {
        // Explicitly using axios (not api instance) to avoid infinite loops
        const { data } = await axios.post(`${API_URL}/refresh`, {
          refresh_token: refreshToken,
        });

        const newAccessToken = data.access_token;
        const newRefreshToken = data.refresh_token;

        // Note: The /refresh endpoint doesn't return the user object.
        // We only update the tokens.
        setAuth({
          accessToken: newAccessToken,
          refreshToken: newRefreshToken,
          user: state.user // keep existing user info
        });

        processQueue(null, newAccessToken);
        originalRequest.headers.Authorization = `Bearer ${newAccessToken}`;
        
        return api(originalRequest);
      } catch (err) {
        processQueue(err, null);
        clearAuth();
        navigate('/login', { replace: true });
        return Promise.reject(err);
      } finally {
        isRefreshing = false;
      }
    }

    return Promise.reject(error);
  }
);

export default api;
