import axios from 'axios';

let logoutCallback = null;

const client = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

function generateCorrelationId() {
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
}

client.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('auth_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    config.headers['X-Correlation-ID'] = generateCorrelationId();
    return config;
  },
  (error) => Promise.reject(error)
);

client.interceptors.response.use(
  (response) => response,
  (error) => {
    if (!error.response) {
      return Promise.reject({
        code: 'NETWORK_ERROR',
        message: 'Unable to reach the server. Check your connection.',
        status: 0,
      });
    }

    const { status, data } = error.response;

    if (status === 401) {
      localStorage.removeItem('auth_token');
      localStorage.removeItem('auth_refresh_token');
      localStorage.removeItem('auth_user');
      if (logoutCallback) {
        logoutCallback();
      }
    }

    const appError = {
      code: data?.code || `HTTP_${status}`,
      message: data?.message || error.message || 'An unexpected error occurred.',
      details: data?.details || null,
      status,
      correlationId: error.response.headers?.['x-correlation-id'] || null,
    };

    return Promise.reject(appError);
  }
);

export function setLogoutCallback(cb) {
  logoutCallback = cb;
}

export default client;
