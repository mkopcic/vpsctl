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
- [x] GET /servers, /servers/ping
- [x] POST /servers/exec, /update, /restart
- [x] GET /jobs?server=&limit= (za web GUI)

---

## Tjedan 7 - SQLite GOTOVO

- [x] modernc.org/sqlite (pure Go, bez CGO)
- [x] database/sql interface
- [x] sync.Once za singleton DB konekciju
- [x] PRAGMA journal_mode=WAL
- [x] CREATE TABLE IF NOT EXISTS
- [x] INSERT, SELECT, Scan
- [x] bool -> int konverzija (SQLite nema bool tip)
- [x] internal/store/store.go + job.go
- [x] vpsctl history [server] --limit
- [x] duration_ms mjerenje po operaciji
- [x] source="cli" vs source="api"

---

## Tjedan 8 - Web GUI + Error Handling GOTOVO

### Go embed package
- [x] //go:embed direktiva — kompajler pakira staticke fajlove u binary
- [x] embed.FS tip
- [x] fs.Sub za stripanje path prefixa
- [x] http.FileSystem interface za Gin StaticFS
- [x] Jedan binary ima sve — Go kod + HTML + CSS + JS

### Web UI (vanilla JS)
- [x] Servers tab — tablica + Ping All s latencijama i bojama
- [x] Execute tab — exec forma + Quick Update/Restart
- [x] History tab — job history iz SQLite, filter po serveru
- [x] API tab — dokumentacija svih endpointa s curl primjerima
- [x] Redirect / -> /ui/ u serve.go
- [x] escapeHtml() za XSS zastitu u JS-u

### Error handling refaktor
- [x] SilenceUsage: true — Cobra ne pokazuje Usage za SSH runtime greske
- [x] runOnServer je void — ne vraca error Cobri, handla prikaz sam
- [x] friendlyError() — mapa poznatih gresaka na jasne poruke:
      - "Interactive authentication required" -> nema sudo permisije
      - "Unit X not found" -> servis ne postoji
      - "command not found" -> komanda nije pronadjena
      - "permission denied" -> permission denied

---

## Tjedan 9 - Docker (sljedece)

### Plan:
- Dockerfile — multi-stage build (builder + minimal image)
- Volume mountanje za data/ i configs/
- Deploy na Hetzner

### Sto treba nauciti:
- Multi-stage Dockerfile u Go-u
- Kako Go binary radi u Alpine containeru
- docker-compose.yml za vpsctl

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
| error handling | GOTOVO | RunE, SilenceUsage, friendlyError |
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
| embed.FS | GOTOVO | internal/web/web.go |
| fs.Sub + http.FileSystem | GOTOVO | web GUI serving |
| interfaces | DJELOMICNO | ssh.AuthMethod |
| pointers | DJELOMICNO | &cobra.Command |
| Docker | NIJE POCETO | tjedan 9 |