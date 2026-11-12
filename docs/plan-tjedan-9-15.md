# vpsctl — plan tjedan 9–15: Docker → Kubernetes

Nastavak na `marijan-ucenje.md`. Redoslijed: prvo Docker, pa Kubernetes.
Channel manager dolazi nakon toga, kao zaseban projekt (skica na dnu).

Princip: svaki infra tjedan nosi sa sobom i **Go koncept** koji ti taj infra
korak ionako nameće. Kubernetes te tjera da napišeš bolji Go — to je poanta.

---

## Spojeni redoslijed (ovaj plan + `stack-komponente.md`)

Dva plana se ne vode paralelno, nego isprepleteno. Svaki korak otključava
sljedeći.

| # | Korak | Iz kojeg plana | Zašto baš tada |
|---|---|---|---|
| 1 | Ostatak Tjedna 0 + API auth + zajednički sloj za CLI i API | ovaj | Malo posla, blokira sve ostalo. |
| 2 | Faza 1–2: inventory, doctor, stack check | stack | Read-only, odmah korisno na pravim serverima. |
| 3 | Tjedan 9–10: Docker + CI | ovaj | CI ionako treba za Fazu 3 (asseti se grade u GitHub Actionsu). Gotenberg = prva stvarna primjena Dockera na produkciji. |
| 4 | Faza 3–4: deploy, rollback, db backup | stack | Treba CI artefakt iz koraka 3. |
| 5 | Tjedan 11–15: Kubernetes na zasebnom k3s VPS-u | ovaj | vpsctl **i** jedan Laravel app u clusteru (vidi Tjedan 11). |
| 6 | Faza 6 (audit) + Tjedan 14 (Prometheus) | oba | Periodični audit izlaže metrike, Grafana ih prikazuje. |
| 7 | Faza 5: stack install recepti | stack | Najveći rizik, najmanja korist — zadnje. |

---

## Tjedan 0 — blokeri prije Dockera

Ovo nisu "nice to have". Svaki od njih ti razbije container.

Stanje provjereno u kodu 12.11.2026.

| # | Problem | Gdje | Zašto puca | Status |
|---|---|---|---|---|
| 1 | `viper.AddConfigPath("./configs")` | `internal/config/config.go` | Relativan path. U containeru config dolazi iz volumea/ConfigMapa, ne iz `./configs`. Treba `VPSCTL_CONFIG` env override. | riješeno (`VPSCTL_CONFIG`) |
| 2 | `dbPath = "data/vpsctl.db"` | `internal/store/store.go` | Isto — relativan. SQLite mora na volume, inače gubiš history pri svakom restartu. | riješeno (`VPSCTL_DATA`) |
| 3 | `logFile = "logs/vpsctl.log"` | `internal/logger/logger.go` | U containeru logovi idu na **stdout**, ne u fajl. Fajl-log je anti-pattern za Docker/K8s. | riješeno (`serve` → stdout, CLI ostaje na fajlu; `VPSCTL_LOG_FILE`) |
| 4 | `return r.Run(addr)` | `cmd/serve.go` | Nema graceful shutdowna. K8s šalje SIGTERM → in-flight SSH exec se prekine nasred posla. | riješeno (`http.Server` + `Shutdown`, 25 s) |
| 5 | `vpsctl.exe` + `vpsctl.exe~` su commitani | root repoa | — | nije bilo točno: binarke nikad nisu ušle u history (repo ima ~50 KB) |
| 6 | **API nema autentikaciju** | `internal/api/api.go`, `cmd/serve.go` | `POST /servers/exec` izvršava proizvoljnu naredbu na bilo kojem serveru, a `serve` sluša na svim sučeljima. Na javnom VPS-u to je otvoren RCE na sve servere. | riješeno (token uvijek obavezan) |
| 7 | CLI i API dupliciraju logiku | `cmd/exec.go`, `cmd/update.go`, `internal/api/api.go` | Update skripta postoji dvaput; svaki sloj sam radi SSH + `store.Save` + log. Svaka nova operacija (deploy, doctor) bi se pisala dvaput. | riješeno (`internal/ops`) |

Za #6:
- [x] Bearer token iz `VPSCTL_API_TOKEN`. Bez njega: na loopbacku se generira
      novi pri svakom startu, na bilo kojoj drugoj adresi `serve` odbija startati.
      Token je obavezan i na localhostu — inače bi bilo koja web stranica u
      browseru mogla poslati POST na `127.0.0.1:8080/servers/exec`.
- [x] Default bind `127.0.0.1`, javno samo uz eksplicitni `--listen` / `VPSCTL_LISTEN`
- [ ] Pristup UI-ju kroz WireGuard ili SSH tunel, ne javni port (promet je HTTP)
- [x] `serve` mod nema interaktivni password prompt (ključ ili password iz YAML-a)

Za #7:
- [x] `internal/ops`: `Runner.Run` radi SSH + history + log; `cmd/` i `internal/api/` ga samo zovu

Dodatno, ne blokira Docker ali blokira produkciju:
- ~~`ssh.InsecureIgnoreHostKey()`~~ — riješeno (`known_hosts` + `vpsctl trust`).
- Passwordi u plain YAML-u. U K8s to ide u Secret, ne u ConfigMap.
- ~~Nula testova~~ — riješeno (`ssh`, `certs`, `cmd`, `config`, `store`, `api`, `ops`).
- ~~`go.mod`: sve ovisnosti označene `// indirect`~~ — riješeno (`go mod tidy`).

**Gotovo kad:** binary radi iz bilo kojeg working directoryja, sve staze dolaze iz env varijabli s razumnim defaultima, a API bez tokena ne odgovara.

---

## Tjedan 9 — Docker

### Cilj
Jedan image, `docker run` radi, podaci prežive restart.

### Taskovi
- [x] Refaktor staza iz Tjedna 0 → `VPSCTL_CONFIG`, `VPSCTL_DATA`, `VPSCTL_PORT`, `VPSCTL_LISTEN`
- [x] Logger u `serve` modu piše na stdout (fajl samo ako je `VPSCTL_LOG_FILE` postavljen)
- [x] Graceful shutdown u `serve.go` — `http.Server` + `signal.NotifyContext` + `srv.Shutdown(ctx)`
- [x] `/healthz` endpoint (uvijek 200) i `/readyz` (200 tek kad je DB otvorena)
- [ ] `.dockerignore` — `.git`, `*.exe`, `logs/`, `data/`, `configs/servers.yaml`
- [ ] Multi-stage `Dockerfile`
- [ ] Verzija kroz ldflags umjesto `const Version` — `-X github.com/mkopcic/vpsctl/cmd.Version=$(git describe --tags)`
- [ ] `docker-compose.yml` — volume za `data/`, bind-mount za `configs/`
- [ ] HEALTHCHECK direktiva
- [ ] Non-root user u imageu
- [ ] SSH ključ i `known_hosts` kao read-only mount, ne u imageu
- [ ] Deploy na Hetzner, ostavi da radi tjedan dana — **tek nakon API autha
      (Tjedan 0, #6)**, port objavljen samo na `127.0.0.1` / WireGuard sučelju
- [ ] Gotenberg kao drugi servis u compose-u (`stack-komponente.md`, §6)

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
- [x] Testovi za `internal/config` i `internal/store`
- [ ] Workflow za jedan Laravel app: `composer install --no-dev`, `npm run build`, artefakt (tar) koji Faza 3 deploya — vidi `stack-komponente.md`

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

### Druga traka: Laravel app u clusteru
Sam vpsctl je tanka K8s vježba (jedna replika, SQLite, alat koji SSH-a prema
van). Uz njega kroz tjedne 11–15 ide i jedan Laravel app
(npr. `laravel-postgresql-demo`):

- [ ] Image: FrankenPHP ili nginx + php-fpm, multi-stage (composer + node build)
- [ ] Deployment za web, zaseban Deployment za `queue:work`
- [ ] CronJob za `schedule:run`
- [ ] Job za `migrate --force` prije rollouta
- [ ] Postgres (StatefulSet ili CloudNativePG), Redis
- [ ] `.env` → ConfigMap + Secret

Ingress, TLS, probe, Helm chart i Argo CD iz sljedećih tjedana rade se na
**oba** appa. Laravel app je onaj koji ide na javni Ingress.

### Gotovo kad
`kubectl port-forward svc/vpsctl 8080:8080` i UI radi iz clustera, a Laravel
app odgovara na svom Serviceu.

---

## Tjedan 12 — Kubernetes: mreža i storage

### Taskovi
- [ ] Ingress (k3s ima Traefik ugrađen) — za Laravel app. vpsctl **ne ide na
      javni Ingress**: pod drži SSH pristup svim serverima, pa ostaje na
      `port-forward` / WireGuard
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
      (vpsctl: egress samo na SSH portove servera iz `servers.yaml`)
- [ ] Audit iz `stack-komponente.md` (Faza 6) kao metrike: dani do isteka
      certifikata, PHP EOL, pending security updatei po serveru
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
| context.Context | gotovo (SSH, CLI) | — |
| graceful shutdown | gotovo (`serve`) | — |
| env config / 12-factor | gotovo | — |
| HTTP middleware (auth) | gotovo (`TokenAuth`) | — |
| testing, table-driven, `httptest` | gotovo | proširiti u tjednu 10 |
| interfaces (pravi) | djelomično | Faza 1 (`Host`, `Panel`, `Component`) + channel manager |
| log/slog | nije počelo | tjedan 14 |
| Prometheus metrics | nije počelo | tjedan 14 |
| errgroup, worker pool | nije počelo | channel manager |
| generics | nije počelo | channel manager |
| Docker | nije počelo | tjedan 9 |
| Kubernetes | nije počelo | tjedan 11–15 |
| Helm | nije počelo | tjedan 13 |
| Argo CD / GitOps | nije počelo | tjedan 15 |
