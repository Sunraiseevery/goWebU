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

## Web UI

The server also hosts a tiny static interface for managing hosts and
starting tunnels. Once running, open `http://localhost:8080/` in a web
browser to access it.

### API Endpoints

- `GET /hosts` list saved hosts
- `POST /hosts` add or update a host
- `POST /start` start a new tunnel (records command history)
- `POST /stop` stop a running tunnel
- `GET /history` list recent command history
