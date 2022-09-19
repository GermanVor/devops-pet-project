# .env

```
ADDRESS="localhost:8080"
REPORT_INTERVAL=(int64)
POLL_INTERVAL=(int64)
STORE_INTERVAL="300s"
STORE_FILE="/tmp/devops-metrics-db.json"
RESTORE="true"
KEY=""
DATABASE_DSN=""
```

`ADDRESS` - (`a` Flag, `Server/Agent`) - Common for Agent and Server Adress (Server Address and Agent requests endpoint Address).

`REPORT_INTERVAL` - (`r` Flag, `Agent`) - The time in seconds when Agent sent `Metrics` to the Server.

`POLL_INTERVAL` - (`p` Flag, `Agent`) - The time in seconds when Agent collects `Metrics`.

`STORE_INTERVAL` - (`i` Flag, `Server`) - The time in seconds after which the current server readings are reset to disk
(value 0 — makes the recording synchronous).

`STORE_FILE` - (`f` Flag, `Server`) - The name of the file in which Server will store `Metrics` (Empty name turn off storing `Metrics`).

`RESTORE` - (`r` Flag, `Server`) - Bool value. `true` - At startup Server will try to load data from `STORE_FILE`. `false` - Server will create new `STORE_FILE` file in startup.

`KEY` - (`r` Flag, `Server/Agent`) - Static key (for educational purposes) for `Metrics` hash generation.
If the `Agent` is started with a `KEY` (`-k=...` or `KEY=""` in `.env`), then all metrics will be subscribed
by simple hash function.

`DATABASE_DSN` - (`r` Flag, `Server`) - Database address to connect server with (For exemple postgres://zzman:@localhost:5432/postgres). If there is `DATABASE_DSN` server will connect with
DataBase (PostgreSQL), create tables and will store metrics there (`STORE_INTERVAL`, `STORE_FILE` and `RESTORE` will be ignored).
