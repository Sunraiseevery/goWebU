# goWebU

[English](README.md) | [中文](README.zh.md)


Simple SSH port forwarding service with SQLite-backed host records and command history.

## Building

```
go build
```

## Running

```
./goWebU -db data.db -addr :8080
```

Or run without building:

```
go run . -db data.db -addr :8080
```

When the server starts, it will attempt to open your default browser to
`http://localhost:8080/`. If it does not open automatically, you can
manually visit the URL.

## Web UI

The server hosts a tiny static interface for managing hosts and starting
tunnels.

### API Endpoints

- `GET /hosts` list saved hosts
- `POST /hosts` add or update a host
- `POST /start` start a new tunnel (records command history)
- `POST /stop` stop a running tunnel
- `GET /history` list recent command history
