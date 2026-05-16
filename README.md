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

| Komanda                      | Opis                                       |
| ---------------------------- | ------------------------------------------ |
| `vpsctl version`             | Ispiši verziju                             |
| `vpsctl list`                | Listaj sve konfigurirane servere           |
| `vpsctl ping`                | TCP ping svih servera s latencijom         |
| `vpsctl exec <server> <cmd>` | Izvrši komandu na jednom serveru           |
| `vpsctl exec all <cmd>`      | Izvrši komandu na svim serverima paralelno |

### Primjeri

```bash
vpsctl list
vpsctl ping
vpsctl exec mellon "uptime"
vpsctl exec mellon "df -h"
vpsctl exec all "uptime"
```

---

## ⚙️ Konfiguracija

Serveri se definiraju u `configs/servers.yaml`.

> ⚠️ `servers.yaml` nije na gitu. Koristi `servers.example.yaml` kao predložak.

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

Prioritet kojim se autentikacija rješava:

1. 🔑 **SSH ključ** — ako je `key:` definiran i postoji na disku
2. 🔒 **Password iz YAML-a** — ako je `password:` definiran
3. ⌨️ **Interaktivni prompt** — pita šifru u terminalu

Podržava `password` i `keyboard-interactive` SSH metode.

---

## 📁 Struktura projekta

```
vpsctl/
├── cmd/
│   ├── root.go        # Cobra root
│   ├── version.go     # vpsctl version
│   ├── list.go        # vpsctl list
│   ├── ping.go        # vpsctl ping — TCP dial, paralelno, latencija
│   └── exec.go        # vpsctl exec
├── internal/
│   ├── config/
│   │   └── config.go  # YAML loader
│   └── ssh/
│       └── ssh.go     # SSH konekcija i auth
├── configs/
│   ├── servers.yaml          # tvoji serveri (nije na gitu)
│   └── servers.example.yaml  # predložak
└── main.go
```

---

## 🗺️ Roadmap

- [x] Cobra CLI setup
- [x] YAML config (Viper)
- [x] SSH exec — jedan server
- [x] SSH exec — svi serveri paralelno
- [x] Password + key + interaktivna autentikacija
- [x] TCP ping s latencijom — paralelno, 3s timeout
- [ ] Logging rezultata u fajl
- [ ] `vpsctl update` — apt update/upgrade
- [ ] `vpsctl restart <servis>` — systemctl restart
- [ ] `vpsctl ssh <server>` — interaktivni shell
