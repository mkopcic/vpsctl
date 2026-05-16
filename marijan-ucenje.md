# Marijan - Go Learning Path

Na temelju originalnog plana i svega sto smo radili zajedno.

---

## Trenutni status

**Projekt:** `vpsctl`
**GitHub:** https://github.com/mkopcic/vpsctl

---

## Tjedan 1 - Osnove + CLI GOTOVO

- [x] package, func, struct
- [x] error handling (RunE, return err pattern)
- [x] goroutine basics
- [x] Cobra CLI — version, list, ping
- [ ] pointers (detaljnije)
- [ ] interfaces (detaljnije)

---

## Tjedan 2 - File Handling + Config GOTOVO

- [x] YAML citanje + Viper
- [x] Struct mapiranje (mapstructure tagovi)
- [x] configs/servers.yaml, internal/config/config.go

---

## Tjedan 3 - SSH GOTOVO

- [x] golang.org/x/crypto/ssh
- [x] Key auth, password auth, keyboard-interactive
- [x] session.CombinedOutput
- [x] Interaktivni prompt (golang.org/x/term)

---

## Tjedan 4 - Parallel Execution GOTOVO

- [x] goroutines + sync.WaitGroup
- [x] goroutine + channel + select (SSH timeout)
- [x] net.DialTimeout, net.JoinHostPort
- [x] time.Since, time.After

---

## Tjedan 5 - Logging GOTOVO (bez Zapa)

- [x] internal/logger/logger.go
- [x] Append u logs/vpsctl.log
- [x] Preskocen Zap — stdlib dovoljan

---

## Tjedan 6 - REST API GOTOVO

- [x] Gin framework (gin.Default, gin.Engine, gin.H)
- [x] Route grupiranje (r.Group)
- [x] GET i POST handleri
- [x] JSON binding (ShouldBindJSON, binding:"required")
- [x] JSON response (c.JSON, http.StatusOK)
- [x] omitempty u struct tagovima
- [x] vpsctl serve --port flag
- [x] GET /servers, /servers/ping
- [x] POST /servers/exec, /update, /restart

---

## Tjedan 7 - SQLite GOTOVO

- [x] modernc.org/sqlite (pure Go, bez CGO)
- [x] database/sql interface
- [x] sync.Once za singleton DB konekciju
- [x] PRAGMA journal_mode=WAL
- [x] CREATE TABLE IF NOT EXISTS
- [x] INSERT, SELECT, Scan u Go-u
- [x] bool -> int konverzija (SQLite nema bool)
- [x] internal/store/store.go — DB init
- [x] internal/store/job.go — Job struct, Save, List, ListByServer
- [x] vpsctl history [server] --limit
- [x] duration_ms mjerenje po operaciji
- [x] source="cli" vs source="api" razlikovanje

---

## Bonus (nije u originalnom planu)

- [x] vpsctl update — apt/yum/apk auto detektira
- [x] vpsctl restart — systemctl + is-active provjera
- [x] Colored output — [OK]/[FAIL]/[INFO]
- [x] IPv6 fix — net.JoinHostPort
- [x] servers.example.yaml + .gitignore

---

## Tjedan 8 - Docker (sljedece)

### Plan:
- Dockerfile za Go binary (multi-stage build)
- Docker compose
- Deploy na Hetzner

### Sto treba nauciti:
- Multi-stage Dockerfile (builder + minimal image)
- Kako Go binary radi u containeru
- Volume mountanje za data/ i configs/

---

## Nakon Go - Big Picture

### Q2 2026 - Docker + Kubernetes
- [ ] Docker images, volumes, networking
- [ ] Kubernetes: pods, deployments, services, ingress, secrets
- [ ] Helm, Argo CD

### Q3 2026 - Terraform + AWS
- [ ] Terraform za Hetzner
- [ ] AWS: VPC, EC2, ALB, RDS, IAM, S3, CloudWatch

### Q4 2026 - AI Orchestration
- [ ] MCP, tool calling, agents, RAG, vector DB
- [ ] AI Sysadmin Agent

---

## Go koncepti - pregled

| Koncept | Status | Gdje |
|---|---|---|
| package, func, struct | GOTOVO | cijeli projekt |
| error handling | GOTOVO | RunE, return err |
| goroutines + WaitGroup | GOTOVO | exec all, ping |
| goroutine + select + channel | GOTOVO | SSH timeout |
| net.DialTimeout, JoinHostPort | GOTOVO | ping |
| time.Since, time.After | GOTOVO | latencija, timeout |
| os.OpenFile, os.ReadFile | GOTOVO | logger, SSH key |
| YAML + Viper | GOTOVO | config |
| Gin HTTP framework | GOTOVO | REST API |
| JSON bind/response | GOTOVO | API handleri |
| database/sql + SQLite | GOTOVO | store/ |
| sync.Once (singleton) | GOTOVO | store/store.go |
| PRAGMA, WAL mode | GOTOVO | SQLite optimizacija |
| interfaces | DJELOMICNO | ssh.AuthMethod |
| pointers | DJELOMICNO | &cobra.Command |
| Docker | NIJE POCETO | tjedan 8 |