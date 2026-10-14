# vpsctl — plan tjedan 9–15: Docker → Kubernetes

Nastavak na `marijan-ucenje.md`. Redoslijed: prvo Docker, pa Kubernetes.
Channel manager dolazi nakon toga, kao zaseban projekt (skica na dnu).

Princip: svaki infra tjedan nosi sa sobom i **Go koncept** koji ti taj infra
korak ionako nameće. Kubernetes te tjera da napišeš bolji Go — to je poanta.

---

## Tjedan 0 — blokeri prije Dockera

Ovo nisu "nice to have". Svaki od njih ti razbije container.

| # | Problem | Gdje | Zašto puca |
|---|---|---|---|
| 1 | `viper.AddConfigPath("./configs")` | `internal/config/config.go` | Relativan path. U containeru config dolazi iz volumea/ConfigMapa, ne iz `./configs`. Treba `VPSCTL_CONFIG` env override. |
| 2 | `dbPath = "data/vpsctl.db"` | `internal/store/store.go` | Isto — relativan. SQLite mora na volume, inače gubiš history pri svakom restartu. |
| 3 | `logFile = "logs/vpsctl.log"` | `internal/logger/logger.go` | U containeru logovi idu na **stdout**, ne u fajl. Fajl-log je anti-pattern za Docker/K8s. |
| 4 | `return r.Run(addr)` | `cmd/serve.go` | Nema graceful shutdowna. K8s šalje SIGTERM → in-flight SSH exec se prekine nasred posla. |
| 5 | `vpsctl.exe` + `vpsctl.exe~` su commitani | root repoa | ~78 MB binarnih fajlova u gitu. `.gitignore` ih lovi tek od sad, ali su već u historyju. |

Dodatno, ne blokira Docker ali blokira produkciju:
- `ssh.InsecureIgnoreHostKey()` u `internal/ssh/ssh.go` — MITM otvoren. Treba `knownhosts.New()`.
- Passwordi u plain YAML-u. U K8s to ide u Secret, ne u ConfigMap.
- Nula testova. Prvi test piši u tjednu 9, ne "kasnije".

**Gotovo kad:** binary radi iz bilo kojeg working directoryja, sve staze dolaze iz env varijabli s razumnim defaultima.

---

## Tjedan 9 — Docker

### Cilj
Jedan image, `docker run` radi, podaci prežive restart.

### Taskovi
- [ ] Refaktor staza iz Tjedna 0 → `VPSCTL_CONFIG`, `VPSCTL_DATA`, `VPSCTL_PORT`
- [ ] Logger piše na stdout (fajl samo ako je `VPSCTL_LOG_FILE` postavljen)
- [ ] Graceful shutdown u `serve.go` — `http.Server` + `signal.NotifyContext` + `srv.Shutdown(ctx)`
- [ ] `/healthz` endpoint (uvijek 200) i `/readyz` (200 tek kad je DB otvorena)
- [ ] `.dockerignore` — `.git`, `*.exe`, `logs/`, `data/`, `configs/servers.yaml`
- [ ] Multi-stage `Dockerfile`
- [ ] Verzija kroz ldflags umjesto `const Version` — `-X github.com/mkopcic/vpsctl/cmd.Version=$(git describe --tags)`
- [ ] `docker-compose.yml` — volume za `data/`, bind-mount za `configs/`
- [ ] HEALTHCHECK direktiva
- [ ] Non-root user u imageu
- [ ] Deploy na Hetzner, ostavi da radi tjedan dana

### Dockerfile — skica
```dockerfile
FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X github.com/mkopcic/vpsctl/cmd.Version=${VERSION}" \
    -o /out/vpsctl .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/vpsctl /vpsctl
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/vpsctl"]
CMD ["serve"]
```

Bitno: `modernc.org/sqlite` je pure Go, pa `CGO_ENABLED=0` prolazi i možeš na
distroless/static. Da si uzeo `mattn/go-sqlite3`, ovdje bi zapeo — dobra odluka.

### Go koncepti
`context.Context`, `signal.NotifyContext`, `os.Getenv` + defaulti, `http.Server` umjesto `r.Run()`, build tagovi i ldflags.

### Gotovo kad
`docker compose up`, otvoriš UI, napraviš exec, `docker compose restart`, history je i dalje tu.

---

## Tjedan 10 — CI, registry, scanning

### Cilj
Push na `main` → image u GHCR-u, skeniran.

### Taskovi
- [ ] GitHub Actions workflow: `go vet`, `go test ./...`, `golangci-lint`, build
- [ ] `docker buildx` multi-arch (amd64 + arm64)
- [ ] Push u `ghcr.io/mkopcic/vpsctl`, tagovi `:sha`, `:vX.Y.Z`, `:latest`
- [ ] Trivy scan imagea, workflow pada na HIGH/CRITICAL
- [ ] `gosec` na kodu
- [ ] Prvi pravi testovi: `internal/config`, `friendlyError()`, `internal/store`

Trivy + gosec ti direktno pokrivaju "nemam formalni SAST/SCA tooling" rupu.
Nakon ovog tjedna imaš konkretan odgovor.

### Go koncepti
`testing` paket, table-driven testovi, `t.TempDir()`, `httptest.NewServer`, interfaces za mockanje SSH-a (ovo ti je u tablici još `DJELOMIČNO`).

---

## Tjedan 11 — Kubernetes: osnove

### Setup
k3s na Hetzner VPS-u (`curl -sfL https://get.k3s.io | sh -`). Jedan node,
~5 EUR/mj, pravi cluster. Lokalno `kind` samo za brze eksperimente.
Ne kreći od managed GKE/EKS — platit ćeš, a naučiti manje.

### Taskovi
- [ ] `kubectl` osnove: `get`, `describe`, `logs`, `exec`, `apply`, `delete`
- [ ] Pod vs ReplicaSet vs Deployment — razumjeti zašto Pod nikad ne pišeš ručno
- [ ] Namespace `vpsctl`
- [ ] Deployment manifest za vpsctl image iz GHCR-a
- [ ] Service (ClusterIP) + `kubectl port-forward` za test
- [ ] ConfigMap za `servers.yaml` **bez** passworda
- [ ] Secret za passworde, mountan kao env ili fajl
- [ ] `imagePullSecret` za privatni GHCR

### Gotovo kad
`kubectl port-forward svc/vpsctl 8080:8080` i UI radi iz clustera.

---

## Tjedan 12 — Kubernetes: mreža i storage

### Taskovi
- [ ] Ingress (k3s ima Traefik ugrađen) — `vpsctl.tvoja-domena.hr`
- [ ] cert-manager + Let's Encrypt, automatski TLS
- [ ] PersistentVolumeClaim za SQLite (`local-path` provisioner)
- [ ] **Deployment → StatefulSet** za SQLite, i razumjeti zašto: SQLite ne podnosi dva writera na isti fajl preko mreže. `replicas: 1` je ovdje jedini točan odgovor.
- [ ] Liveness / readiness / startup probe na `/healthz` i `/readyz`
- [ ] Resource requests i limits (bez njih te scheduler slaže nasumično)

### Ovdje ćeš naletjeti na zid
SQLite + Kubernetes = ne skalira. To je *feature* ovog tjedna, ne bug.
Zabilježi u plan: "za horizontalno skaliranje treba Postgres". To je ujedno
prirodan most prema channel manageru, gdje Postgres ionako treba.

---

## Tjedan 13 — Helm

### Taskovi
- [ ] `helm create vpsctl` pa izbaciti 80% scaffolda
- [ ] Templatirati: image tag, replicas, resources, ingress host, storage size
- [ ] `values.yaml` + `values-prod.yaml`
- [ ] Secret kroz `--set` ili external secret, nikad u `values.yaml` na gitu
- [ ] `helm lint`, `helm template`, `helm upgrade --install`
- [ ] Chart u `charts/` unutar repoa

### Gotovo kad
`helm upgrade --install vpsctl ./charts/vpsctl -f values-prod.yaml` deploya cijelu stvar iz nule.

---

## Tjedan 14 — operacije i sigurnost

Ovaj tjedan je najviše vrijedan za infrastructure security smjer.

### Taskovi
- [ ] ServiceAccount + RBAC — Role, RoleBinding, princip najmanje privilegije
- [ ] `securityContext`: `runAsNonRoot`, `readOnlyRootFilesystem`, `drop: [ALL]` capabilities
- [ ] NetworkPolicy — default deny, pa eksplicitno otvoriti što treba
- [ ] Pod Security Standards (`restricted`)
- [ ] Prometheus + Grafana (kube-prometheus-stack preko Helma)
- [ ] `/metrics` endpoint u vpsctl-u — `prometheus/client_golang`, brojači za exec/update/restart, histogram trajanja
- [ ] Strukturirano logiranje: zamijeni `internal/logger` s `log/slog` u JSON modu
- [ ] Grafana dashboard: broj jobova, error rate, p95 trajanje

### Go koncepti
`log/slog`, Prometheus collectori, middleware pattern u Ginu, `context` kroz cijeli call stack.

---

## Tjedan 15 — GitOps

### Taskovi
- [ ] Argo CD na cluster
- [ ] Repo `vpsctl-gitops` — Argo Application pokazuje na Helm chart
- [ ] Auto-sync, self-heal
- [ ] Image updater ili CI koji bumpa tag u gitops repou
- [ ] Rollback vježba — pokvari deploy namjerno, vrati ga kroz Argo

### Gotovo kad
Mijenjaš kod → merge → Argo sam deploya → vidiš u Grafani. Nijedan `kubectl apply` ručno.

---

## Nakon toga — channel manager

Zaseban projekt, ne dio vpsctl-a. Skica da znaš kamo ide.

**Što je:** sinkronizacija ARI (Availability, Rates, Inventory) između PMS-a i
OTA kanala (Booking.com, Expedia, Airbnb), plus povlačenje rezervacija natrag.

**MVP — "mini channel manager", bez pravih API pristupa:**
- Dva lažna OTA adaptera iza jednog `OTAAdapter` interfacea (jedan JSON, jedan XML — namjerno različiti, da interface ima smisla)
- Mapiranje: tvoj room type + rate plan → OTA-ov room ID + rate ID
- Push ARI promjena, delta umjesto full sync
- Pull rezervacija, idempotentno po external ID-u
- Worker queue s retryjem i eksponencijalnim backoffom
- Rate limiting po kanalu (`golang.org/x/time/rate`)
- Outbox pattern — nikad ne gubi promjenu ako je OTA dolje
- Detekcija overbookinga

**Stack:** Postgres + `sqlc`, `river` ili `asynq` za queue, `chi` ili Echo, `errgroup`, Docker + K8s (sve iz tjedana 9–15).

**Go koncepti koje tek ovdje stvarno naučiš:** interfaces kao arhitektura (ne samo `ssh.AuthMethod`), worker pooli, `errgroup`, generics, pošten error wrapping s `errors.Is`/`errors.As`.

Ovo je i domena Rentlia — ako tamo prođe, ovaj projekt ti je najjači mogući govor. Ako ne prođe, i dalje je najbolji Go projekt koji možeš staviti u portfolio.

---

## Ažurirana tablica Go koncepata

| Koncept | Status | Kad |
|---|---|---|
| context.Context | nije počelo | tjedan 9 |
| graceful shutdown | nije počelo | tjedan 9 |
| env config / 12-factor | nije počelo | tjedan 9 |
| testing, table-driven | nije počelo | tjedan 10 |
| interfaces (pravi) | djelomično | tjedan 10 + channel manager |
| log/slog | nije počelo | tjedan 14 |
| Prometheus metrics | nije počelo | tjedan 14 |
| errgroup, worker pool | nije počelo | channel manager |
| generics | nije počelo | channel manager |
| Docker | nije počelo | tjedan 9 |
| Kubernetes | nije počelo | tjedan 11–15 |
| Helm | nije počelo | tjedan 13 |
| Argo CD / GitOps | nije počelo | tjedan 15 |
