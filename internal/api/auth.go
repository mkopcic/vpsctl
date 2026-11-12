package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// TokenAuth pusta samo requeste s headerom "Authorization: Bearer <token>".
//
// API izvrsava proizvoljne komande na svim serverima, pa je token obavezan i
// na localhostu: bez njega bi bilo koja web stranica otvorena u browseru
// mogla poslati POST na http://127.0.0.1:8080/servers/exec.
func TokenAuth(token string) gin.HandlerFunc {
	want := []byte(token)

	return func(c *gin.Context) {
		got, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")

		// Prazan token nikad ne prolazi — ni kad je Setup pozvan bez njega.
		// ConstantTimeCompare: trajanje usporedbe ne otkriva koliko je
		// znakova pogodjeno.
		if !ok || len(want) == 0 || subtle.ConstantTimeCompare([]byte(got), want) != 1 {
			c.Header("WWW-Authenticate", `Bearer realm="vpsctl"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "nedostaje ili je neispravan API token"})
			return
		}

		c.Next()
	}
}

// NewToken generira nasumican token (256 bita, hex).
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// IsLoopback javlja slusa li adresa samo na ovom racunalu.
// Prazan host ("" ili ":8080") znaci sva sucelja — to nije loopback.
func IsLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
