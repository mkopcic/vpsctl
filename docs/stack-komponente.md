# vpsctl — stack komponente i deploy Laravel appova

Smjer nakon osnovnog vpsctl-a: alat koji stvarno koristim na svojim serverima
(Virtualmin, CWP), a ne samo vježba. vpsctl prestaje biti "SSH exec na više
servera" i postaje alat koji **zna što je na serveru, što appu treba, i kako
to dovesti u red**.

Veza s ostalim planovima:
- `marijan-ucenje.md` — tjedni 1–7, osnove (gotovo)
- `plan-tjedan-9-15.md` — Docker → Kubernetes. Ovaj dokument je **paralelna
  grana**: Tjedan 0 (blokeri) vrijedi i ovdje, ostalo se može raditi
  prije, poslije ili isprepleteno.

Princip ostaje isti: svaka faza nosi **Go koncept** koji ti ionako nameće.

---

## 1. Princip: deklarativni zahtjevi po appu

U `configs/apps.yaml` svaki app navodi što treba. vpsctl uspoređuje to sa
stvarnim stanjem servera i (tek na zahtjev) instalira ono što fali.

```yaml
apps:
  - name: rezervacije
    server: mellon
    panel_user: mellonwe          # vlasnik domene u panelu
    path: /home/mellonwe/apps/rezervacije
    php: "8.3"
    db:
      engine: mariadb             # mariadb | mysql | pgsql | sqlite
      name: rezervacije
    requires: [mariadb, redis, imagick, chrome, node:22]
    workers:
      - name: queue
        command: artisan queue:work --sleep=3 --tries=3
      - name: horizon
        command: artisan horizon
```

| Komanda | Što radi |
|---|---|
| `vpsctl stack check <app>` | Read-only. Za svaki `requires` pokrene Detect i ispiše OK / fali / kriva verzija. |
| `vpsctl stack install <app>` | Instalira što fali. Default je **dry-run** — ispiše plan, tek `--apply` izvršava. |
| `vpsctl doctor <server>` | Matrica cijelog servera (vidi dolje). |

### Komponenta = recept
Svaka komponenta ima tri metode, s varijantama po **OS obitelji**
(debian / rhel) i **panelu** (virtualmin / cwp / none):

- **Detect** — je li instalirano, koja verzija. Nikad ništa ne mijenja.
- **Install** — idempotentno. Drugi poziv ne smije ništa pokvariti.
- **Verify** — dokaz da radi (ne samo "paket postoji").

### Prije svakog deploya
```bash
<php> /usr/local/bin/composer check-platform-reqs
```
**S PHP binaryjem tog appa.** Hvata sve ekstenzije koje fale (imagick, redis,
pgsql, intl, gd...). Ako padne, deploy staje prije nego išta dira.

---

## 2. PHP binary po appu

Na multi-PHP serverima `php` u PATH-u je gotovo uvijek **kriva verzija**.
Panel adapter vraća točan put:

| Panel | Primjer |
|---|---|
| CWP | `/opt/alt/php83/usr/bin/php` |
| Virtualmin (ondrej/Remi) | `/usr/bin/php8.3` |

Composer i artisan **uvijek** preko njega:
```bash
<php> /usr/local/bin/composer install --no-dev --optimize-autoloader
<php> artisan migrate --force
```

U kodu: nijedna komanda ne smije sadržavati golo `php`. To je pravilo koje
treba imati test.

---

## 3. Baze

| Engine | Pravilo |
|---|---|
| MySQL / MariaDB | Panel već ima jednu. **Nikad ne instalirati obje** (konflikt porta, socketa, paketa). Baza i korisnik kroz panel (`virtualmin create-database`, CWP API). |
| PostgreSQL | Virtualmin ga podržava. CWP ne → instalirati iz PGDG repoa. Bind `127.0.0.1`, auth `scram-sha-256`, **jedan role po appu**. |
| SQLite | Samo `pdo_sqlite` + `sqlite3` CLI. Datoteka **izvan** `releases/`, u `shared/database/`. WAL mode. |

### `vpsctl db create|backup|restore|list <app>`

| Engine | Backup |
|---|---|
| mysql / mariadb | `mysqldump --single-transaction` |
| pgsql | `pg_dump -Fc` |
| sqlite | `sqlite3 db ".backup ..."` — **NIKAD `cp` dok app piše** |

- Automatski backup prije `migrate` u svakom deployu.
- Offsite: `restic` na Hetzner Storage Box ili S3, s retencijom.

### GUI za baze
- **Preporuka:** lokalni klijent preko SSH tunela, ništa na serveru.
  DBeaver CE ili Beekeeper Studio (besplatni, MySQL + PG + SQLite).
- `vpsctl db gui <server> [app]` — otvori SSH tunel. Po potrebi privremeno
  pokrene **Adminer** (jedna PHP datoteka, MySQL + PG + SQLite) na
  `127.0.0.1` servera i **ugasi ga i obriše na izlazu**. Nema trajno
  izloženog panela.
- phpMyAdmin već dolazi s CWP-om i Virtualminom. Ako ostaje → iza IP
  whitelista ili basic autha.
- pgAdmin 4 je težak (Python) — ne na produkciju.

---

## 4. Cache / queue

- **Redis ili Valkey** (BSD fork, paketi u novijim distroima).
  `bind 127.0.0.1`, `requirepass`, `maxmemory` + `allkeys-lru` za cache.
- Izolacija appova: zaseban `REDIS_PREFIX` i/ili DB broj. Kritični appovi →
  zasebna instanca.
- `phpredis` ekstenzija **za PHP verziju appa**.
- **OPcache** produkcijski: `validate_timestamps=0`. Posljedica: deploy
  **MORA** napraviti reload FPM-a nakon swapa symlinka, inače servira stari kod.
- Queue workeri, Horizon, Reverb, Inertia SSR: supervisor ili systemd unit po
  appu, **generiran iz `apps.yaml`** (`text/template`).

---

## 5. Imagick

- ImageMagick + `imagick` ekstenzija **za svaku PHP verziju zasebno**
  (CWP: `pecl` za alt-php; Virtualmin: paket `php8.x-imagick`).
- Provjeriti `policy.xml` — PDF/PS su po defaultu zabranjeni. Otvoriti samo
  ako app to stvarno treba.
- Verify:
  ```bash
  <php> -r "echo Imagick::getVersion()['versionString'];"
  ```

---

## 6. Headless Chrome (Browsershot / PDF)

| Opcija | Kad | Kako |
|---|---|---|
| **A — Gotenberg** (preporuka za PDF) | Kad treba samo HTML → PDF | Docker, `127.0.0.1:3000`. Izoliran, bez Chrome ovisnosti na hostu, isti na svakom distrou. |
| **B — Browsershot** | Kad app već koristi Browsershot | Node + puppeteer + `chrome-headless-shell` u homeu app usera. Recept instalira i sistemske libove (nss, atk, gbm...). |

Verify za obje: generirati testni PDF i provjeriti da nije prazan.

Opcija A je usput i prva stvarna primjena Dockera iz `plan-tjedan-9-15.md`.

---

## 7. Node / npm / Composer

- Assete (`npm run build`) graditi u **CI-ju** (GitHub Actions) i deployati
  artefakt. Server tada ne treba Node samo zbog Vitea.
- Ako Node ipak treba (SSR, Browsershot): verzija po appu iz `.nvmrc`, preko
  `fnm` u homeu app usera ili sistemski iz NodeSource repoa.
- Composer 2 globalno (`/usr/local/bin/composer`), `self-update` u `audit`-u.

### `vpsctl doctor <server>`
```
PHP        8.1   8.2   8.3
imagick     -     ✓     ✓
redis       ✓     ✓     ✓
pgsql       -     -     ✓
intl        ✓     ✓     ✓

MariaDB 10.11 ✓   PostgreSQL 16 ✓   Redis 7.2 ✓
Node 22 (fnm: rezervacije)   Composer 2.8 ✓
Gotenberg 8 ✓   supervisor ✓
```

---

## 8. Kako tome pristupiti u kodu

### Prvo: preduvjeti u postojećem kodu

| # | Problem | Gdje | Zašto ovdje puca |
|---|---|---|---|
| 1 | `RunCommand` otvara **novu SSH konekciju za svaku komandu** | `internal/ssh/ssh.go` | `doctor` pokreće 30–50 komandi. Treba trajan `*ssh.Client` po serveru i nova `Session` po komandi. |
| 2 | `ssh.InsecureIgnoreHostKey()` | `internal/ssh/ssh.go` | Alat koji **instalira pakete kao root** ne smije prihvatiti bilo koji host. `knownhosts.New()`. |
| 3 | Timeout preko `time.After`, nema `context` | `internal/ssh/ssh.go` | Install zna trajati minutama, a Ctrl+C mora prekinuti čisto. `RunCommand(ctx, ...)`. |
| 4 | Samo `CombinedOutput` | `internal/ssh/ssh.go` | Detect treba odvojeno stdout / stderr / exit code. |
| 5 | Relativni pathovi, nema testova | Tjedan 0 iz `plan-tjedan-9-15.md` | Isto vrijedi ovdje. |

### Arhitektura — interfacei
Ovo je mjesto gdje interfacei postaju **arhitektura**, a ne samo
`ssh.AuthMethod`.

```go
// internal/host — sve što komponenta smije raditi sa serverom
type Host interface {
    Run(ctx context.Context, cmd string) (Result, error)
    Facts() Facts // OS obitelj, panel, PHP verzije — skupljeno jednom
}

type Result struct {
    Stdout, Stderr string
    ExitCode       int
}

// internal/panel — razlike Virtualmin / CWP / none
type Panel interface {
    Name() string
    PHPBinary(app config.App) (string, error)
    CreateDatabase(ctx context.Context, app config.App) error
}

// internal/stack — jedan recept
type Component interface {
    Name() string
    Detect(ctx context.Context, h Host, req Requirement) (State, error)
    Install(ctx context.Context, h Host, req Requirement) (Plan, error)
    Verify(ctx context.Context, h Host, req Requirement) error
}
```

Zašto ovako:
- **`Host` je interface** → testovi rade s lažnim hostom koji vraća snimljeni
  output (`dpkg -l`, `php -m`...). Nijedan test ne treba pravi server.
- **`Install` vraća `Plan`** (listu komandi), ne izvršava ih. Dry-run je onda
  besplatan: ispiši plan. `--apply` ga izvrši. Isti plan ide u job history.
- **Registry** komponenti: `map[string]Component`, svaka se registrira u
  `init()`. `requires: [node:22]` se parsira u `Requirement{Name:"node", Version:"22"}`.

### Struktura
```
internal/
├── host/        Host interface, SSH implementacija, fake za testove
├── facts/       detekcija OS-a, panela, PHP verzija (inventory)
├── panel/       virtualmin.go, cwp.go, none.go
├── stack/       registry + jedan fajl po komponenti
│   ├── mariadb.go, pgsql.go, sqlite.go
│   ├── redis.go, imagick.go, chrome.go
│   └── node.go, composer.go, phpext.go
├── deploy/      releases/, shared/, symlink swap, FPM reload, rollback
└── db/          create, backup, restore, gui tunel
configs/
└── apps.yaml
```

---

## 9. Redoslijed

Svaka faza je zatvorena cjelina koja je korisna sama za sebe.

### Faza 1 — inventory (read-only)
- [x] Preduvjeti 1–4 iz tablice gore (SSH refaktor + `vpsctl trust`)
- [ ] `Host` interface + SSH implementacija + fake
- [ ] `facts`: OS obitelj (`/etc/os-release`), panel (postoji li `virtualmin` / `/usr/local/cwpsrv`), PHP verzije
- [ ] `configs/apps.yaml` + učitavanje
- [ ] Detect za sve komponente iz ovog dokumenta

**Go koncepti:** interfacei kao arhitektura, `context.Context`, `os/exec`-stil
rezultat (stdout/stderr/exit), table-driven testovi s fake hostom, `testdata/` fixture fajlovi.

**Gotovo kad:** `vpsctl inventory mellon` ispiše OS, panel i PHP verzije, bez ijedne promjene na serveru.

### Faza 2 — doctor + stack check
- [ ] Matrica PHP verzije × ekstenzije
- [ ] `stack check <app>` — usporedba `requires` sa stanjem
- [ ] Izlaz i kao JSON (`--json`) → odmah i API endpoint i tab u Web UI-ju

**Go koncepti:** `text/tabwriter`, `encoding/json`, worker pool (paralelni Detect po komponentama i serverima), `errgroup`.

### Faza 3 — app deploy / rollback
- [ ] `releases/<timestamp>`, `shared/` (`.env`, `storage/`, `database/`)
- [ ] `composer check-platform-reqs` s PHP-om appa — prvi korak, staje ako padne
- [ ] Artefakt iz GitHub Actionsa (assete se ne grade na serveru)
- [ ] Backup baze prije `migrate`
- [ ] Atomični swap symlinka + **FPM reload** (OPcache)
- [ ] Restart workera
- [ ] `vpsctl rollback <app>` — symlink na prethodni release
- [ ] Zadržati zadnjih N releaseova

**Go koncepti:** pošten error wrapping (`fmt.Errorf("...: %w")`, `errors.Is`), "korak + undo" pattern, `text/template` za supervisor/systemd unite.

### Faza 4 — db backup / restore + db gui
- [ ] `db backup|restore|list` po engineu
- [ ] restic offsite + retencija
- [ ] `db gui` — SSH local port forward (`client.Listen` / `net.Listen` + `io.Copy`), privremeni Adminer koji se briše na izlazu (`defer` + `signal.NotifyContext`)

**Go koncepti:** `io.Copy` i streamanje (dump ne smije ići kroz memoriju), port forwarding s goroutinama, cleanup na signal.

### Faza 5 — stack install recepti
- [ ] Install + Verify za svaku komponentu, debian i rhel varijante
- [ ] Dry-run default, `--apply` za izvršavanje
- [ ] Svaki install zapisan u job history s planom i outputom

**Go koncepti:** generics (npr. zajednički helper za "detektiraj → planiraj → izvrši → verificiraj"), build matrica u testovima.

### Faza 6 — audit / health
- [ ] `composer self-update`, verzije paketa, istekli certifikati
- [ ] Redis/DB bind na `127.0.0.1`, `requirepass` postavljen, phpMyAdmin zaštićen
- [ ] Periodično iz `serve` moda (`time.Ticker`), rezultat u Web UI i kasnije Prometheus (`plan-tjedan-9-15.md`, Tjedan 14)

---

## 10. Pravila koja vrijede kroz sve faze

- Detect **nikad** ne mijenja server.
- Install je **idempotentan** i po defaultu dry-run.
- Nikad golo `php` — uvijek binary appa.
- Nikad dvije MySQL obitelji na istom serveru.
- Nijedan servis (DB, Redis, Gotenberg, Adminer) ne sluša na javnoj adresi.
- Svaka komponenta ima test s fake hostom prije nego što se pokrene na pravom serveru.
- Prvo na testnom VPS-u (Hetzner), tek onda na mellonu.
