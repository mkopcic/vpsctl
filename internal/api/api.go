package api

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mkopcic/vpsctl/internal/config"
	"github.com/mkopcic/vpsctl/internal/logger"
	sshclient "github.com/mkopcic/vpsctl/internal/ssh"
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
	Server string `json:"server"`
	Output string `json:"output"`
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"` // omitempty = ne pojavljuje se u JSON-u ako je prazan
}

// updateScript detektira package manager automatski
const updateScript = `
if command -v apt-get >/dev/null 2>&1; then
    apt-get update && apt-get upgrade -y
elif command -v yum >/dev/null 2>&1; then
    yum update -y
elif command -v apk >/dev/null 2>&1; then
    apk update && apk upgrade
else
    echo "unknown package manager"
    exit 1
fi
`

// Setup registrira sve rute — poziva se iz cmd/serve.go
func Setup(r *gin.Engine) {
	servers := r.Group("/servers")
	{
		servers.GET("", listServers)        // GET  /servers
		servers.GET("/ping", pingServers)   // GET  /servers/ping
		servers.POST("/exec", execCommand)  // POST /servers/exec
		servers.POST("/update", updateServers) // POST /servers/update
		servers.POST("/restart", restartService) // POST /servers/restart
	}
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

	var list []ServerInfo
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
	var results []PingResult

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

	results := runOnTargets(cfg, req.Server, req.Command)
	c.JSON(http.StatusOK, results)
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

	results := runOnTargets(cfg, req.Server, updateScript)
	c.JSON(http.StatusOK, results)
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

	cfg, err := config.Load()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	command := fmt.Sprintf("systemctl restart %s && systemctl is-active %s", req.Service, req.Service)
	results := runOnTargets(cfg, req.Server, command)
	c.JSON(http.StatusOK, results)
}

// --- HELPER ---

// runOnTargets izvrsava komandu na jednom serveru ili svima ("all")
// Vraca listu ServerResult — ista logika kao u cmd/exec.go ali za API
func runOnTargets(cfg *config.Config, target, command string) []ServerResult {
	var results []ServerResult

	for _, s := range cfg.Servers {
		// Preskoči ako nije traženi server (osim ako je "all")
		if target != "all" && s.Name != target {
			continue
		}

		out, err := sshclient.RunCommand(sshclient.Config{
			User:     s.User,
			Host:     s.Host,
			Port:     s.Port,
			Key:      s.Key,
			Password: s.Password,
		}, command)

		if err != nil {
			results = append(results, ServerResult{
				Server: s.Name,
				OK:     false,
				Error:  err.Error(),
			})
		} else {
			logger.Log(s.Name, command, out)
			results = append(results, ServerResult{
				Server: s.Name,
				Output: out,
				OK:     true,
			})
		}

		// Ako je tražen jedan server, stani
		if target != "all" {
			break
		}
	}

	return results
}
