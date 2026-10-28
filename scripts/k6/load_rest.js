import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '10s', target: 20 },  // Разгон: от 0 до 20 виртуальных юзеров за 10 сек
    { duration: '30s', target: 50 },  // Держим стабильную нагрузку 50 юзеров 30 сек
    { duration: '10s', target: 0 },   // Плавный спад нагрузки
  ],
  thresholds: {
    http_req_duration: ['p(95)<150', 'p(99)<300'], // 95% запросов должны отвечать быстрее 150мс
    http_req_failed: ['rate<0.01'],                 // Ошибок должно быть меньше 1%
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  // 1. Загрузка главной страницы
  const resHome = http.get(`${BASE_URL}/`);
  check(resHome, { 'home page status 200': (r) => r.status === 200 });

  // 2. Регистрация уникального пользователя
  const uniqueName = `load_${__VU}_${Date.now()}_${Math.floor(Math.random() * 1000)}`;
  const payload = JSON.stringify({
    username: uniqueName,
    email: `${uniqueName}@test.com`, // <--- ДОБАВЬ ВОТ ЭТУ СТРОЧКУ
    password: 'Password123!',
  });
  const headers = { headers: { 'Content-Type': 'application/json' } };

  const resReg = http.post(`${BASE_URL}/api/v1/auth/register`, payload, headers);
  check(resReg, { 'registered 201': (r) => r.status === 201 || r.status === 200 });

  // 3. Авторизация (получение JWT)
  const resLogin = http.post(`${BASE_URL}/api/v1/auth/login`, payload, headers);
  const loginSuccess = check(resLogin, { 'logged in 200': (r) => r.status === 200 });

  // 4. Запрос списка чатов с токеном
  if (loginSuccess && resLogin.json('token')) {
    const token = resLogin.json('token');
    const authHeaders = {
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
    };

    const resChats = http.get(`${BASE_URL}/api/v1/chats`, authHeaders);
    check(resChats, { 'get chats 200': (r) => r.status === 200 });
  }

  sleep(0.1); // Небольшая пауза 100 мс между действиями
}