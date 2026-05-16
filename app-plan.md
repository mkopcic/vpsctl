# vpsctl - app plan

CLI alat za upravljanje VPS serverima. Pisan u Go-u kao learning path za Go
od strane senior Laravel/PHP + sysadmin profila.

---

## Sto je napravljeno

### Infrastruktura

- [x] Cobra CLI setup - `cmd/root.go`, entry point `main.go`
- [x] YAML config loader - `internal/config/config.go` (Viper)
- [x] SSH client - `internal/ssh/ssh.go` (golang.org/x/crypto/ssh)
- [x] Color output - `internal/output/output.go` (ANSI, bez dependencyja)
- [x] Logger - `internal/logger/logger.go` (append u logs/vpsctl.log)
- [x] `.gitignore` - `servers.yaml`, `vpsctl.exe`, `logs/` isključeni
- [x] `configs/servers.example.yaml` - template

### Komande

- [x] `vpsctl version`
- [x] `vpsctl list` - serveri iz YAML-a
- [x] `vpsctl ping` - TCP dial, paralelno, 3s timeout, latencija u ms
- [x] `vpsctl exec <server|all> <cmd>` - SSH exec, paralelno za all
- [x] `vpsctl update <server|all>` - apt/yum/apk, detektira automatski
- [x] `vpsctl restart <server|all> <servis>` - systemctl restart + is-active provjera

### Auth

- [x] SSH key auth
- [x] Password iz YAML-a (password + keyboard-interactive metode)
- [x] Interaktivni prompt kao fallback

### Tehnicke stvari

- [x] SSH connection timeout - 10s
- [x] SSH exec timeout - 30s (goroutine + select)
- [x] IPv6 kompatibilnost - net.JoinHostPort umjesto fmt.Sprintf
- [x] Logging svih exec/update/restart operacija

---

## Sto nedostaje

### vpsctl ssh - interaktivni shell

```bash
vpsctl ssh mellon
```

Otvori interaktivni SSH shell kroz vpsctl config.
Zahtijeva PTY alokaciju - kompleksnije od exec.

```go
session.RequestPty("xterm", 80, 40, ssh.TerminalModes{})
session.Shell()
```

Treba i stdin/stdout/stderr spojiti na terminal:

```go
session.Stdin = os.Stdin
session.Stdout = os.Stdout
session.Stderr = os.Stderr
```

---

## Key-based auth - plan

Za automatizaciju bez upisa sifre, jednom po serveru:

```bash
ssh-copy-id -i ~/.ssh/id_rsa.pub -p 2222 user@server.hr
```

Moguca komanda `vpsctl setup-key <server>` koja to automatizira.

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
│   └── restart.go
├── internal/
│   ├── config/config.go
│   ├── ssh/ssh.go
│   ├── output/output.go
│   └── logger/logger.go
├── configs/
│   ├── servers.yaml
│   └── servers.example.yaml
├── main.go
├── go.mod
├── .gitignore
└── app-plan.md
```

---

## Go koncepti nauceni kroz vpsctl

| Koncept | Gdje |
|---|---|
| Cobra CLI | `cmd/` |
| Viper config | `internal/config/` |
| Structs | `config.Server`, `sshclient.Config` |
| Error handling | `RunE`, `return err` pattern |
| Goroutines + WaitGroup | `exec all`, `ping` |
| Packages / moduli | `internal/ssh`, `internal/config`, `internal/output`, `internal/logger` |
| `net.DialTimeout` | TCP ping |
| `net.JoinHostPort` | IPv4/IPv6 kompatibilnost |
| `time.Since` | mjerenje latencije |
| goroutine + select + time.After | SSH exec timeout |
| `os.OpenFile` | append logging |
| `golang.org/x/crypto/ssh` | SSH konekcija i auth |
| `session.CombinedOutput` | capture SSH outputa |