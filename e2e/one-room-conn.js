import ws from 'k6/ws';
import { check } from 'k6';

export const options = {
  stages: [
    { duration: '10s', target: 1000 },
    { duration: '10s', target: 10000 },
    { duration: '10s', target: 10000 },
  ],
};

export default function () {
  const url = 'ws://localhost:8080';
  const path = '/e2e-room';

  const res = ws.connect(url + path, socket => {
    socket.on('error', e => {
      if (e.error() != 'websocket: close sent') {
        console.log('An unexpected error occured: ', e.error());
      }
    });
  });

  check(res, { 'status is 101': (r) => r && r.status === 101 });
}
