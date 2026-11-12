package api

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mkopcic/vpsctl/internal/config"
	"github.com/mkopcic/vpsctl/internal/ops"
	"github.com/mkopcic/vpsctl/internal/store"
)

// --- REQUEST STRUKTURE ---
// Gin ih automatski puni iz JSON body-a (binding:"required" = 400 ako fali)

type ExecRequest struct {
	Server  string `json:"server" binding:"required"`
	Command string `json:"command" binding:"required"`
}

type TargetRequest struct {
	Server string `json:"server" binding:"required"`
}

type RestartRequest struct {
	Server  string `json:"server" binding:"required"`
	Service string `json:"service" binding:"required"`
}

// --- RESPONSE STRUKTURA ---
// Svaki handler vraca listu ovih, jedan po serveru

type ServerResult struct {
	Server   string `json:"server"`
	Output   string `json:"output"`
	OK       bool   `json:"ok"`
	ExitCode int    `json:"exit_code"`
	Error    string `json:"error,omitempty"` // omitempty = ne pojavljuje se u JSON-u ako je prazan
}

// apiRunner zapisuje jobove sa source="api". Bez interaktivnog prompta —
// server nema terminal na kojem bi netko upisao password.
var apiRunner = ops.Runner{Source: "api", NoPrompt: true}

// Setup registrira sve rute — poziva se iz cmd/serve.go.
// Sve API rute traze token (vidi auth.go). /healthz i /readyz ne traze:
// Docker i Kubernetes probe ne salju Authorization header.
func Setup(r *gin.Engine, token string) {
	r.GET("/healthz", healthz)
	r.GET("/readyz", readyz)

	auth := r.Group("", TokenAuth(token))

	servers := auth.Group("/servers")
	{
		servers.GET("", listServers)             // GET  /servers
		servers.GET("/ping", pingServers)        // GET  /servers/ping
		servers.POST("/exec", execCommand)       // POST /servers/exec
		servers.POST("/update", updateServers)   // POST /servers/update
		servers.POST("/restart", restartService) // POST /servers/restart
	}

	// GET /jobs?server=mellon&limit=20 — job history za web GUI
	auth.GET("/jobs", listJobs)
}

// --- HANDLERI ---

// GET /servers — vraca listu svih servera iz YAML-a
func listServers(c *gin.Context) {
	cfg, err := config.Load()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Vracamo samo sigurne podatke, bez passworda i kljuca
	type ServerInfo struct {
		Name string `json:"name"`
		Host string `json:"host"`
		Port int    `json:"port"`
		User string `json:"user"`
	}

	// Prazan array umjesto null ako nema servera — UI radi list.map(...)
	list := []ServerInfo{}
	for _, s := range cfg.Servers {
		list = append(list, ServerInfo{
			Name: s.Name,
			Host: s.Host,
			Port: s.Port,
			User: s.User,
		})
	}

	c.JSON(http.StatusOK, list)
}

// GET /servers/ping — TCP ping svih servera, vraca latenciju
func pingServers(c *gin.Context) {
	cfg, err := config.Load()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type PingResult struct {
		Server    string `json:"server"`
		Host      string `json:"host"`
		OK        bool   `json:"ok"`
		LatencyMs int64  `json:"latency_ms,omitempty"`
	}

	// Ping je brz pa ne trebamo goroutine ovdje, radimo sekvencijalno
	// Ako zelis paralelno, ista logika kao u cmd/ping.go s WaitGroup
	results := []PingResult{}

	for _, s := range cfg.Servers {
		addr := net.JoinHostPort(s.Host, strconv.Itoa(s.Port))

		start := time.Now()
		conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
		latency := time.Since(start)

		if err != nil {
			results = append(results, PingResult{Server: s.Name, Host: s.Host, OK: false})
			continue
		}
		conn.Close()

		results = append(results, PingResult{
			Server:    s.Name,
			Host:      s.Host,
			OK:        true,
			LatencyMs: latency.Milliseconds(),
		})
	}

	c.JSON(http.StatusOK, results)
}

// POST /servers/exec — izvrsi komandu na jednom ili svim serverima
// Body: {"server": "mellon", "command": "uptime"}
// Body: {"server": "all",    "command": "df -h"}
func execCommand(c *gin.Context) {
	var req ExecRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cfg, err := config.Load()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	runOnTargets(c, cfg, req.Server, req.Command, ops.ExecTimeout)
}

// POST /servers/update — system update na jednom ili svim serverima
// Body: {"server": "mellon"}
// Body: {"server": "all"}
func updateServers(c *gin.Context) {
	var req TargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cfg, err := config.Load()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	runOnTargets(c, cfg, req.Server, ops.UpdateScript, ops.UpdateTimeout)
}

// POST /servers/restart — restart systemd servisa
// Body: {"server": "mellon", "service": "nginx"}
// Body: {"server": "all",    "service": "php8.3-fpm"}
func restartService(c *gin.Context) {
	var req RestartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	command, err := ops.RestartCommand(req.Service)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cfg, err := config.Load()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	runOnTargets(c, cfg, req.Server, command, ops.RestartTimeout)
}

// --- HELPER ---

// runOnTargets izvrsava komandu na jednom serveru ili svima ("all") i salje
// JSON odgovor. SSH, history i log radi ops.Runner.
// Context je onaj HTTP requesta — ako klijent prekine request, prekida se i SSH.
func runOnTargets(c *gin.Context, cfg *config.Config, target, command string, timeout time.Duration) {
	servers, err := ops.Targets(cfg, target)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Prazan array umjesto null ako nema servera
	results := []ServerResult{}

	for _, s := range servers {
		runCtx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		res := apiRunner.Run(runCtx, s, command)
		cancel()

		sr := ServerResult{
			Server:   res.Server,
			Output:   res.Output,
			OK:       res.OK(),
			ExitCode: res.ExitCode,
		}
		if res.Err != nil {
			sr.Error = res.Err.Error()
		}
		results = append(results, sr)
	}

	c.JSON(http.StatusOK, results)
}

// GET /healthz — proces je ziv. Uvijek 200.
func healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GET /readyz — spreman za promet tek kad je baza otvorena.
func readyz(c *gin.Context) {
	if err := store.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// GET /jobs?server=mellon&limit=20
// Vraca job history iz SQLite za web GUI history tab
func listJobs(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	server := c.Query("server")

	var jobs []store.Job
	var err error

	if server != "" {
		jobs, err = store.ListByServer(server, limit)
	} else {
		jobs, err = store.List(limit)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Vrati prazan array umjesto null ako nema jobova
	if jobs == nil {
		jobs = []store.Job{}
	}

	c.JSON(http.StatusOK, jobs)
}
