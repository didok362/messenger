import ws from 'k6/ws';
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '10s', target: 15 }, // Разгон: подключаем 15 ботов
    { duration: '25s', target: 30 }, // Держим 30 постоянных онлайн-собеседников
    { duration: '5s', target: 0 },   // Плавное отключение
  ],
  thresholds: {
    'checks': ['rate>0.95'], // 95%+ успешных проверок
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
// Автоматически переключаем http -> ws или https -> wss
const WS_URL = BASE_URL.replace('https://', 'wss://').replace('http://', 'ws://');

// 1. Подготовка: регистрируем общего собеседника "Боба", которому все будут писать
export function setup() {
  const targetName = `bob_target_${Date.now()}`;
  const payload = JSON.stringify({
    username: targetName,
    email: `${targetName}@test.com`,
    password: 'Password123!',
  });
  const headers = { headers: { 'Content-Type': 'application/json' } };

  http.post(`${BASE_URL}/api/v1/auth/register`, payload, headers);
  const res = http.post(`${BASE_URL}/api/v1/auth/login`, payload, headers);
  
  return {
    targetUserID: res.json('user.id'),
  };
}

// 2. Поведение каждого виртуального пользователя
export default function (data) {
  // А) Бот регистрируется и получает свой токен
  const botName = `ws_bot_${__VU}_${__ITER}_${Date.now()}`;
  const payload = JSON.stringify({
    username: botName,
    email: `${botName}@test.com`,
    password: 'Password123!',
  });
  const headers = { headers: { 'Content-Type': 'application/json' } };

  http.post(`${BASE_URL}/api/v1/auth/register`, payload, headers);
  const loginRes = http.post(`${BASE_URL}/api/v1/auth/login`, payload, headers);
  const token = loginRes.json('token');
  if (!token) return;

  const authHeaders = {
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
  };

  // Б) Создаем диалог с Бобом
  const chatRes = http.post(
    `${BASE_URL}/api/v1/chats/direct`,
    JSON.stringify({ target_user_id: data.targetUserID }),
    authHeaders
  );
  const chatID = chatRes.json('id');
  if (!chatID) return;

  // В) Подключаемся к WebSocket и держим сокет открытым
  const url = `${WS_URL}/api/v1/ws?token=${token}`;

  const res = ws.connect(url, {}, function (socket) {
    socket.on('open', () => {
      // Каждые 2 секунды отправляем сообщение прямо в WebSocket
      socket.setInterval(() => {
        socket.send(JSON.stringify({
          chat_id: chatID,
          content: `Привет от бота ${__VU}! [${Date.now()}]`,
        }));
      }, 2000);
    });

    socket.on('message', (msg) => {
      // Сообщение прилетело по сокету
    });

    // Держим сессию активной 20 секунд, потом закрываем
    socket.setTimeout(() => {
      socket.close();
    }, 20000);
  });

  check(res, { 'websocket handshake 101': (r) => r && r.status === 101 });

  sleep(1);
}