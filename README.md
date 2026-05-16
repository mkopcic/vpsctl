# 🖥️ vpsctl

> CLI alat za upravljanje VPS serverima, pisan u Go-u.

---

## ⚡ Quick start

```bash
git clone https://github.com/mkopcic/vpsctl
cd vpsctl
cp configs/servers.example.yaml configs/servers.yaml
go build -o vpsctl.exe .
```

---

## 📋 Komande

| Komanda                                 | Opis                                        |
| --------------------------------------- | ------------------------------------------- |
| `vpsctl version`                        | Ispisi verziju                              |
| `vpsctl list`                           | Listaj sve konfigurirane servere            |
| `vpsctl ping`                           | TCP ping svih servera s latencijom          |
| `vpsctl exec <server\|all> <cmd>`       | Izvrsi komandu na jednom ili svim serverima |
| `vpsctl update <server\|all>`           | System update (apt/yum/apk)                 |
| `vpsctl restart <server\|all> <servis>` | Restart systemd servisa                     |

### 📌 Primjeri

```bash
vpsctl list
vpsctl ping
vpsctl exec mellon "uptime"
vpsctl exec all "df -h"
vpsctl update mellon
vpsctl update all
vpsctl restart mellon nginx
vpsctl restart all php8.3-fpm
```

---

## ⚙️ Konfiguracija

Serveri se definiraju u `configs/servers.yaml`.

> servers.yaml nije na gitu. Koristi `servers.example.yaml` kao predlozak.

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

## 🔐 Autentikacija

1. SSH kljuc ako je `key:` definiran i postoji na disku
2. Password iz YAML-a ako je `password:` definiran
3. Interaktivni prompt ako nema ni jedno od gore navedenog

---

## 📄 Logging

Svaki `exec`, `update` i `restart` logira se u `logs/vpsctl.log`:

```
[2026-05-16 12:49:53] mellon - "uptime"
 12:49:53 up 23 days, 4:54, 0 users
---
```

---

## 📁 Struktura projekta

```
vpsctl/
├── cmd/
│   ├── root.go
│   ├── version.go
│   ├── list.go
│   ├── ping.go
│   ├── exec.go
│   ├── update.go
│   └── restart.go
├── internal/
│   ├── config/config.go
│   ├── ssh/ssh.go
│   ├── output/output.go
│   └── logger/logger.go
├── configs/
│   ├── servers.yaml
│   └── servers.example.yaml
└── main.go
```

---

## 🗺️ Roadmap

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
- [ ] vpsctl ssh - interaktivni shell
