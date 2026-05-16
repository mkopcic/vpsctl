# Marijan - Go Learning Path

Na temelju originalnog plana i svega sto smo radili zajedno.

---

## Trenutni status

**Projekt:** `vpsctl`
**GitHub:** https://github.com/mkopcic/vpsctl

---

## Tjedan 1 - Osnove + CLI GOTOVO

### Go koncepti nauceni:
- [x] package
- [x] func
- [x] struct
- [x] error handling (RunE, return err pattern)
- [x] goroutine basics
- [ ] pointers (koristeni implicitno, treba detaljnije)
- [ ] interfaces (treba detaljnije)

### Prakticni zadaci:
- [x] `vpsctl version`
- [x] `vpsctl list`
- [x] `vpsctl ping`
- [x] Cobra CLI setup

---

## Tjedan 2 - File Handling + Config GOTOVO

### Nauceno:
- [x] YAML citanje
- [x] Viper config loader
- [x] Struct mapiranje iz YAML-a (mapstructure tagovi)

### Prakticni zadaci:
- [x] `configs/servers.yaml` s name, host, port, user, key, password
- [x] `internal/config/config.go`
- [x] `list` i `ping` citaju iz YAML-a

---

## Tjedan 3 - SSH GOTOVO

### Nauceno:
- [x] golang.org/x/crypto/ssh
- [x] SSH key auth
- [x] SSH password auth
- [x] SSH keyboard-interactive auth
- [x] session.CombinedOutput za capture outputa
- [x] Interaktivni password prompt (golang.org/x/term)

### Prakticni zadaci:
- [x] `vpsctl exec mellon "uptime"`
- [x] `internal/ssh/ssh.go`

---

## Tjedan 4 - Parallel Execution GOTOVO

### Nauceno:
- [x] goroutines (go func(){})
- [x] sync.WaitGroup
- [x] goroutine + channel + select (SSH timeout)
- [x] time.After za timeout
- [x] net.DialTimeout za TCP ping
- [x] net.JoinHostPort (IPv4/IPv6 kompatibilnost)
- [x] time.Since za mjerenje latencije

### Prakticni zadaci:
- [x] `vpsctl exec all "uptime"` - svi serveri paralelno
- [x] `vpsctl ping` - TCP dial, latencija u ms
- [x] SSH connection timeout 10s, exec timeout 30s

---

## Tjedan 5 - Logging GOTOVO (bez Zapa)

### Sto smo napravili:
- [x] `internal/logger/logger.go`
- [x] Append u `logs/vpsctl.log`
- [x] Format: timestamp + server + komanda + output
- [x] Logira exec, update, restart (CLI i API)

### Preskoceno svjesno:
- Zap logger (stdlib je dovoljan za sada)

---

## Tjedan 6 - REST API GOTOVO

### Nauceno:
- [x] Gin framework (gin.Default, gin.Engine)
- [x] Route grupiranje (r.Group("/servers"))
- [x] GET i POST handleri
- [x] JSON binding (ShouldBindJSON, binding:"required")
- [x] JSON response (c.JSON, http.StatusOK)
- [x] gin.H{} za brze JSON odgovore
- [x] omitempty tag u struct-u
- [x] Gin Logger i Recovery middleware
- [x] --port flag u Cobra komandi
- [x] Gin vs net/http razlika

### Prakticni zadaci:
- [x] `vpsctl serve` - pokrece Gin server
- [x] `GET  /servers` - lista servera (bez passworda!)
- [x] `GET  /servers/ping` - TCP ping svih, JSON s latencijom
- [x] `POST /servers/exec` - SSH exec, jedan ili all
- [x] `POST /servers/update` - system update
- [x] `POST /servers/restart` - systemctl restart

---

## Bonus (nije u originalnom planu)

- [x] `vpsctl update` - apt/yum/apk auto detektira distro
- [x] `vpsctl restart` - systemctl restart + is-active provjera
- [x] Colored output - [OK] zeleno, [FAIL] crveno, [INFO] zuto
- [x] `internal/output/output.go` - reusable color helpers
- [x] servers.example.yaml + .gitignore
- [x] IPv6 fix - net.JoinHostPort umjesto fmt.Sprintf

---

## Tjedan 7 - SQLite (sljedece)

### Plan:
- Koristiti modernc.org/sqlite ili mattn/go-sqlite3
- `database/sql` interface

### Ideje za vpsctl:
- job history - kad, koji server, sto, rezultat, trajanje
- server inventory umjesto YAML-a (ili uz YAML)
- scheduled jobs

### Sto treba nauciti:
- database/sql interface u Go-u
- CRUD operacije
- Migracije

---

## Tjedan 8 - Docker (uskoro)

### Plan:
- Dockerfile za Go binary (multi-stage build)
- Docker compose
- Deploy na Hetzner

---

## Nakon Go - Big Picture

### Q2 2026 - Docker + Kubernetes
- [ ] Docker images, volumes, networking
- [ ] Kubernetes: pods, deployments, services, ingress, secrets
- [ ] Helm
- [ ] Argo CD

### Q3 2026 - Terraform + AWS
- [ ] Terraform za Hetzner (VPS creation, DNS, firewall, backups)
- [ ] AWS: VPC, EC2, ALB, RDS, IAM, S3, CloudWatch

### Q4 2026 - AI Orchestration
- [ ] MCP, tool calling, agents
- [ ] RAG, vector DB
- [ ] AI Sysadmin Agent

---

## Go koncepti - pregled

| Koncept | Status | Gdje |
|---|---|---|
| package, func, struct | GOTOVO | cijeli projekt |
| error handling | GOTOVO | RunE, return err |
| goroutines | GOTOVO | exec all, ping |
| sync.WaitGroup | GOTOVO | exec all, ping |
| goroutine + select + channel | GOTOVO | SSH timeout |
| net.DialTimeout | GOTOVO | ping |
| time.Since, time.After | GOTOVO | latencija, timeout |
| os.OpenFile | GOTOVO | logger |
| os.ReadFile | GOTOVO | SSH key |
| YAML + Viper | GOTOVO | config |
| Gin HTTP framework | GOTOVO | REST API |
| JSON bind/response | GOTOVO | API handleri |
| interfaces | DJELOMICNO | ssh.AuthMethod |
| pointers | DJELOMICNO | &cobra.Command |
| SQLite | NIJE POCETO | tjedan 7 |
| Docker | NIJE POCETO | tjedan 8 |