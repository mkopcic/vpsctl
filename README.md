# vpsctl

> CLI alat za upravljanje VPS serverima, pisan u Go-u.

---

## Quick start

```bash
git clone https://github.com/mkopcic/vpsctl
cd vpsctl
cp configs/servers.example.yaml configs/servers.yaml
go build -o vpsctl.exe .
```

---

## CLI komande

| Komanda | Opis |
|---|---|
| `vpsctl version` | Ispisi verziju |
| `vpsctl list` | Listaj sve konfigurirane servere |
| `vpsctl ping` | TCP ping svih servera s latencijom |
| `vpsctl exec <server\|all> <cmd>` | Izvrsi komandu na jednom ili svim serverima |
| `vpsctl update <server\|all>` | System update (apt/yum/apk) |
| `vpsctl restart <server\|all> <servis>` | Restart systemd servisa |
| `vpsctl serve` | Pokreni HTTP API server (default port 8080) |

### Primjeri

```bash
vpsctl list
vpsctl ping
vpsctl exec mellon "uptime"
vpsctl exec all "df -h"
vpsctl update mellon
vpsctl restart mellon nginx
vpsctl serve
vpsctl serve --port 9000
```

---

## REST API

Pokreni server:

```bash
vpsctl serve --port 8080
```

| Metoda | Endpoint | Opis |
|---|---|---|
| GET | `/servers` | Lista svih servera (bez passworda i kljuca) |
| GET | `/servers/ping` | TCP ping svih servera s latencijom |
| POST | `/servers/exec` | Izvrsi komandu na jednom ili svim serverima |
| POST | `/servers/update` | System update (apt/yum/apk) |
| POST | `/servers/restart` | Restart systemd servisa |

### Primjeri

```bash
# Lista servera
curl http://localhost:8080/servers

# Ping svih
curl http://localhost:8080/servers/ping

# Exec na jednom serveru
curl -X POST http://localhost:8080/servers/exec \
  -H "Content-Type: application/json" \
  -d '{"server":"mellon","command":"uptime"}'

# Exec na svim serverima
curl -X POST http://localhost:8080/servers/exec \
  -H "Content-Type: application/json" \
  -d '{"server":"all","command":"df -h"}'

# Update jednog servera
curl -X POST http://localhost:8080/servers/update \
  -H "Content-Type: application/json" \
  -d '{"server":"mellon"}'

# Restart servisa
curl -X POST http://localhost:8080/servers/restart \
  -H "Content-Type: application/json" \
  -d '{"server":"mellon","service":"nginx"}'
```

### JSON response format

```json
// GET /servers
[{"name":"mellon","host":"mellon.hr","port":2222,"user":"mellonwe"}]

// GET /servers/ping
[{"server":"mellon","host":"mellon.hr","ok":true,"latency_ms":61}]

// POST /servers/exec (uspjeh)
[{"server":"mellon","output":"13:48 up 23 days...","ok":true}]

// POST /servers/exec (greska)
[{"server":"mellon","ok":false,"error":"timeout nakon 30s"}]
```

---

## Konfiguracija

Serveri se definiraju u `configs/servers.yaml`.

> `servers.yaml` nije na gitu. Koristi `servers.example.yaml` kao predlozak.

```yaml
servers:
  - name: mellon
    host: mellon.hr
    port: 2222
    user: mellonwe
    password: tajna

  - name: web-server
    host: example.com
    port: 22
    user: root
    key: C:/Users/ime/.ssh/id_rsa
```

---

## Autentikacija

1. SSH kljuc ako je `key:` definiran i postoji na disku
2. Password iz YAML-a ako je `password:` definiran
3. Interaktivni prompt ako nema ni jedno od gore navedenog

---

## Logging

Svaki `exec`, `update` i `restart` (CLI i API) logira se u `logs/vpsctl.log`:

```
[2026-05-16 13:48:14] mellon - "uptime"
 13:48:14 up 23 days, 5:53, 0 users
---
```

---

## Struktura projekta

```
vpsctl/
├── cmd/
│   ├── root.go
│   ├── version.go
│   ├── list.go
│   ├── ping.go
│   ├── exec.go
│   ├── update.go
│   ├── restart.go
│   └── serve.go
├── internal/
│   ├── config/config.go
│   ├── ssh/ssh.go
│   ├── output/output.go
│   ├── logger/logger.go
│   └── api/api.go
├── configs/
│   ├── servers.yaml
│   └── servers.example.yaml
└── main.go
```

---

## Roadmap

- [x] Cobra CLI setup
- [x] YAML config
- [x] SSH exec jedan server
- [x] SSH exec svi serveri paralelno
- [x] Password + key + interaktivna autentikacija
- [x] TCP ping s latencijom
- [x] Colored output
- [x] Logging u fajl
- [x] SSH timeout
- [x] vpsctl update
- [x] vpsctl restart
- [x] REST API (Gin)
- [ ] vpsctl ssh - interaktivni shell
- [ ] SQLite - job history
- [ ] Docker deploy