import axios from 'axios';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080/api';

export const api = axios.create({
  baseURL: API_URL,
  timeout: 10000,
});

export const getRiskEvents = (params?: { severity?: string; limit?: number }) => {
  return api.get('/risks', { params });
};

export const getStats = () => {
  return api.get('/stats');
};
