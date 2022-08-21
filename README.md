# .env

```
ADDRESS="localhost:8080"
REPORT_INTERVAL=(int64)
POLL_INTERVAL=(int64)
STORE_INTERVAL="300s"
STORE_FILE="/tmp/devops-metrics-db.json"
RESTORE="true"
```

`ADDRESS` - Common for Agent and Server Adress (Server Address and Agent requests endpoint Address).

`REPORT_INTERVAL` - The time in seconds when Agent sent Metrics to the Server.

`POLL_INTERVAL` - The time in seconds when Agent collects Metrics.

`STORE_INTERVAL` - The time in seconds after which the current server readings are reset to disk
(value 0 — makes the recording synchronous).

`STORE_FILE` - The name of the file in which Server will store Metrics (Empty name turn off storing Metrics).

`RESTORE` - Bool value. `true` - At startup Server will try to load data from `STORE_FILE`. `false` - Server will create new `STORE_FILE` file in startup.
