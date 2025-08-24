# goWebU

[English](README.md) | [中文](README.zh.md)


Simple SSH port forwarding service with SQLite-backed host records and command history.
Supports forwarding multiple local ports to different remote targets within a single session.

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
tunnels. Each session may forward multiple ports simultaneously.

### API Endpoints

- `GET /hosts` list saved hosts
- `POST /hosts` add or update a host
- `POST /start` start a new tunnel. Example payload:

```json
{
  "host_id": 1,
  "forwards": [
    {"lport": 9000, "rhost": "localhost", "rport": 5432},
    {"lport": 9001, "rhost": "localhost", "rport": 5433}
  ]
}
```
- `POST /stop` stop a running tunnel
- `GET /history` list recent command history
