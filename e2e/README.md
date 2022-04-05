# lightcable e2e test

## Depend

- [k6](https://k6.io/)

## Prepare

```bash
ulimit -n 1000000
```

## Test

```bash
k6 run one-room-conn.js
k6 run multi-room-conn.js
k6 run one-room-msg.js
```

