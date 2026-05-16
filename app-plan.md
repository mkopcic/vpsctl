# vpsctl — app plan

CLI alat za upravljanje VPS serverima. Pisan u Go-u kao learning path za Go
od strane senior Laravel/PHP + sysadmin profila.

---

## Što je napravljeno

### Infrastruktura

- [x] Cobra CLI setup — `cmd/root.go`, entry point `main.go`
- [x] YAML config loader — `internal/config/config.go` (Viper, kao Laravel config/)
- [x] SSH client — `internal/ssh/ssh.go` (golang.org/x/crypto/ssh)
- [x] `.gitignore` — `servers.yaml` i `vpsctl.exe` isključeni iz gita
- [x] `configs/servers.example.yaml` — template za onboarding

### Komande

- [x] `vpsctl version` — ispiši verziju
- [x] `vpsctl list` — listaj servere iz YAML-a
- [x] `vpsctl ping` — TCP dial na host:port, paralelno, 3s timeout, latencija u ms
- [x] `vpsctl exec <server> <komanda>` — SSH, izvrši komandu na jednom serveru
- [x] `vpsctl exec all <komanda>` — izvrši na svim serverima paralelno (goroutines)

### Auth

- [x] SSH key auth (key: u YAML-u)
- [x] Password auth iz YAML-a (password: u YAML-u) — password + keyboard-interactive
- [x] Interaktivni prompt — pita šifru ako nema ni key ni password u YAML-u

---

## Što nedostaje

### Logging

`vpsctl exec` rezultati idu samo na stdout — nema zapisa.
Treba log fajl, format: timestamp + server + komanda + output.

```
[2026-05-16 12:49] mellon — uptime
 12:49:53 up 23 days, 4:54, 0 users, load average: 0.18, 0.20, 0.14
```

Go: `os.OpenFile` + `log` package ili `zerolog`.
Kao Laravel Log::info() ali bez dependencyja.

---

### Update komanda

```bash
vpsctl update mellon
vpsctl update all
```

Wrapper oko `exec` s predefiniranom komandom.
Treba detektirati distro (apt vs yum vs apk).

---

### Restart servisa

```bash
vpsctl restart mellon nginx
vpsctl restart all nginx
```

Wrapper oko `exec` — `systemctl restart <servis>`.

---

### SSH shell (bonus)

```bash
vpsctl ssh mellon
```

Interaktivni SSH shell kroz vpsctl config.
Zahtijeva PTY alokaciju — kompleksnije od exec.

---

### Tehničke stvari koje treba srediti

- [ ] Timeout za SSH exec — trenutno može visiti beskonačno
- [ ] Colored output — [OK] zeleno, [FAIL] crveno
- [ ] README.md encoding — riješeno (UTF-8)

---

## Key-based auth — plan

Za automatizaciju bez upisa šifre, jednom po serveru:

```bash
ssh-copy-id -i ~/.ssh/id_rsa.pub -p 2222 user@server.hr
```

Nakon toga u YAML maknuti `password:`, dodati `key:`.

Moguća komanda `vpsctl setup-key <server>` koja to automatizira.

---

## Struktura projekta

```
vpsctl/
├── cmd/
│   ├── root.go        # Cobra root, inicijalizacija
│   ├── version.go     # vpsctl version
│   ├── list.go        # vpsctl list
│   ├── ping.go        # TCP ping, paralelno, latencija
│   └── exec.go        # vpsctl exec <server|all> <cmd>
├── internal/
│   ├── config/
│   │   └── config.go  # YAML loader (Viper)
│   └── ssh/
│       └── ssh.go     # SSH konekcija, auth, exec
├── configs/
│   ├── servers.yaml         # privatno, nije na gitu
│   └── servers.example.yaml # predložak
├── main.go
├── go.mod
├── .gitignore
└── app-plan.md
```

---

## Go koncepti naučeni kroz vpsctl

| Koncept | Gdje |
|---|---|
| Cobra CLI | `cmd/` — kao Artisan commands |
| Viper config | `internal/config/` — kao Laravel config/ |
| Structs + interfaces | `config.Server`, `sshclient.Config` |
| Error handling | `RunE`, `return err` pattern |
| Goroutines + WaitGroup | `exec all`, `ping` — paralelni SSH i TCP |
| Packages / moduli | `internal/ssh`, `internal/config` |
| `net.DialTimeout` | TCP ping s timeoutom |
| `time.Since` | mjerenje latencije |
| `golang.org/x/crypto/ssh` | SSH konekcija i auth |
| `session.CombinedOutput` | capture SSH outputa |