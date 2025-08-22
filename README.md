# goWebU

Simple SSH port forwarding service with SQLite-backed host records and command history.

## Building

```
go build
```

## Running

```
./goWebU -db data.db -addr :8080
```

### API Endpoints

- `GET /hosts` list saved hosts
- `POST /hosts` add or update a host
- `POST /start` start a new tunnel (records command history)
- `POST /stop` stop a running tunnel
- `GET /history` list recent command history
