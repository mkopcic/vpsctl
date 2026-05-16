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
| `vpsctl history [server]` | Prikazi job history iz baze |
| `vpsctl serve` | Pokreni HTTP API server + Web UI |

### Primjeri

```bash
vpsctl list
vpsctl ping
vpsctl exec mellon "uptime"
vpsctl exec all "df -h"
vpsctl update mellon
vpsctl restart mellon nginx
vpsctl history
vpsctl history mellon --limit 50
vpsctl serve
vpsctl serve --port 9000
```

---

## Web UI

```bash
vpsctl serve
# otvori http://localhost:8080
```

Embedded u binary — nema eksternih fajlova. Cetiri taba:

- **Servers** — tablica svih servera, Ping All s latencijama
- **Execute** — SSH exec, Quick Update, Quick Restart
- **History** — job history iz SQLite, filter po serveru
- **API** — dokumentacija svih endpointa s curl primjerima

---

## REST API

| Metoda | Endpoint | Body / Params | Opis |
|---|---|---|---|
| GET | `/servers` | — | Lista svih servera |
| GET | `/servers/ping` | — | TCP ping s latencijom |
| POST | `/servers/exec` | `{"server":"mellon","command":"uptime"}` | SSH exec |
| POST | `/servers/update` | `{"server":"all"}` | System update |
| POST | `/servers/restart` | `{"server":"mellon","service":"nginx"}` | Restart servisa |
| GET | `/jobs` | `?server=mellon&limit=20` | Job history |

---

## Konfiguracija

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

> `servers.yaml` nije na gitu. Koristi `servers.example.yaml` kao predlozak.

---

## Autentikacija

1. SSH kljuc ako je `key:` definiran
2. Password iz YAML-a
3. Interaktivni prompt ako nema ni jedno

---

## Error handling

Poznate greske se prevode u jasne poruke:

| Greska sa servera | Prikazano |
|---|---|
| `Interactive authentication required` | nema sudo permisije |
| `Unit X not found` | servis ne postoji na serveru |
| `command not found` | komanda nije pronadjena |
| `permission denied` | permission denied |

Za sudo gresku — na serveru kao root:
```bash
echo "user ALL=(ALL) NOPASSWD: /bin/systemctl" >> /etc/sudoers.d/vpsctl
```

---

## Logging i history

- `logs/vpsctl.log` — plain text log svake operacije
- `data/vpsctl.db` — SQLite, trajanje, source (cli/api)

```bash
vpsctl history mellon --limit 20
```

---

## Struktura projekta

```
vpsctl/
├── cmd/
│   ├── root.go, version.go, list.go
│   ├── ping.go, exec.go, update.go
│   ├── restart.go, history.go, serve.go
├── internal/
│   ├── config/config.go
│   ├── ssh/ssh.go
│   ├── output/output.go
│   ├── logger/logger.go
│   ├── store/store.go + job.go
│   ├── api/api.go
│   └── web/
│       ├── web.go
│       └── static/ (index.html, style.css, app.js)
├── configs/
│   ├── servers.yaml (nije na gitu)
│   └── servers.example.yaml
├── data/    (nije na gitu)
├── logs/    (nije na gitu)
└── main.go
```

---

## Roadmap

- [x] Cobra CLI setup
- [x] YAML config
- [x] SSH exec + parallel
- [x] Password + key + interaktivna auth
- [x] TCP ping s latencijom
- [x] Colored output
- [x] Logging u fajl
- [x] SSH timeout
- [x] vpsctl update
- [x] vpsctl restart
- [x] REST API (Gin)
- [x] SQLite job history
- [x] Web GUI (embed)
- [x] Error handling (SilenceUsage, friendlyError)
- [ ] Docker deploy
- [ ] vpsctl ssh — interaktivni shell