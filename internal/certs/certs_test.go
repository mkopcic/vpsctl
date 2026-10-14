package certs

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// tlsServer pokrece lokalni HTTPS server. Njegov cert vrijedi za
// 127.0.0.1 i example.com, a potpisan je testnim CA-om iz httptest paketa.
func tlsServer(t *testing.T) (*httptest.Server, *x509.CertPool) {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(srv.Close)

	roots := x509.NewCertPool()
	roots.AddCert(srv.Certificate())
	return srv, roots
}

func TestCheck(t *testing.T) {
	srv, roots := tlsServer(t)
	notAfter := srv.Certificate().NotAfter
	addr := srv.Listener.Addr().String() // 127.0.0.1:port

	tests := []struct {
		name    string
		checker Checker
		domain  string
		wantErr func(error) bool
		wantNeg bool // DaysLeft < 0
	}{
		{
			name:    "valjan",
			checker: Checker{Roots: roots},
			domain:  addr,
			wantErr: func(err error) bool { return err == nil },
		},
		{
			name:    "istekao",
			checker: Checker{Roots: roots, Now: func() time.Time { return notAfter.Add(48 * time.Hour) }},
			domain:  addr,
			wantErr: func(err error) bool {
				e, ok := errors.AsType[x509.CertificateInvalidError](err)
				return ok && e.Reason == x509.Expired
			},
			wantNeg: true,
		},
		{
			name:    "nepoznat izdavatelj",
			checker: Checker{Roots: x509.NewCertPool()},
			domain:  addr,
			wantErr: func(err error) bool {
				_, ok := errors.AsType[x509.UnknownAuthorityError](err)
				return ok
			},
		},
		{
			name:    "krivi hostname",
			checker: Checker{Roots: roots},
			domain:  strings.Replace(addr, "127.0.0.1", "localhost", 1),
			wantErr: func(err error) bool {
				_, ok := errors.AsType[x509.HostnameError](err)
				return ok
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.checker.Check(t.Context(), tt.domain)

			if !tt.wantErr(res.Err) {
				t.Errorf("neocekivana greska: %v", res.Err)
			}
			if !res.NotAfter.Equal(notAfter) {
				t.Errorf("NotAfter = %v, want %v — datum mora biti popunjen i za nevaljan cert", res.NotAfter, notAfter)
			}
			if (res.DaysLeft < 0) != tt.wantNeg {
				t.Errorf("DaysLeft = %d, wantNeg = %v", res.DaysLeft, tt.wantNeg)
			}
		})
	}
}

// newCert generira cert potpisan s parent/parentKey (ili self-signed ako je parent nil)
func newCert(t *testing.T, tmpl *x509.Certificate, parent *x509.Certificate, parentKey *ecdsa.PrivateKey) (*x509.Certificate, *ecdsa.PrivateKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if parent == nil {
		parent, parentKey = tmpl, key
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, parent, &key.PublicKey, parentKey)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return cert, key
}

// TestCheck_IncompleteChain: root -> intermediate -> leaf, ali server salje samo leaf.
// Go verifier s nasim rootom to odbije (UnknownAuthority), a s intermediateom
// u Roots poolu verifikacija prolazi — tada mora uhvatiti incompleteChain.
func TestCheck_IncompleteChain(t *testing.T) {
	now := time.Now()
	ca := func(cn string, serial int64) *x509.Certificate {
		return &x509.Certificate{
			SerialNumber:          big.NewInt(serial),
			Subject:               pkix.Name{CommonName: cn},
			NotBefore:             now.Add(-time.Hour),
			NotAfter:              now.Add(24 * time.Hour),
			IsCA:                  true,
			BasicConstraintsValid: true,
			KeyUsage:              x509.KeyUsageCertSign,
		}
	}

	root, rootKey := newCert(t, ca("Test Root", 1), nil, nil)
	inter, interKey := newCert(t, ca("Test Intermediate", 2), root, rootKey)
	leaf, leafKey := newCert(t, &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}, inter, interKey)

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{{
		Certificate: [][]byte{leaf.Raw}, // samo leaf — intermediate fali
		PrivateKey:  leafKey,
	}}}
	srv.StartTLS()
	t.Cleanup(srv.Close)
	addr := srv.Listener.Addr().String()

	// Kao Linux/curl: zna samo root, intermediate ne moze naci
	roots := x509.NewCertPool()
	roots.AddCert(root)
	res := Checker{Roots: roots}.Check(t.Context(), addr)
	if _, ok := errors.AsType[x509.UnknownAuthorityError](res.Err); !ok {
		t.Errorf("samo root: err = %v, want UnknownAuthorityError", res.Err)
	}

	// Kao Windows/browser: intermediate je nekako dostupan pa verifikacija prolazi
	withInter := x509.NewCertPool()
	withInter.AddCert(inter)
	res = Checker{Roots: withInter}.Check(t.Context(), addr)
	if !errors.Is(res.Err, ErrIncompleteChain) {
		t.Errorf("s intermediateom: err = %v, want ErrIncompleteChain", res.Err)
	}
}

func TestCheck_ConnectionRefused(t *testing.T) {
	srv, roots := tlsServer(t)
	addr := srv.Listener.Addr().String()
	srv.Close()

	res := Checker{Roots: roots, Timeout: time.Second}.Check(t.Context(), addr)

	if res.Err == nil {
		t.Fatal("ocekivana greska za zatvoren port")
	}
	if !res.NotAfter.IsZero() {
		t.Error("NotAfter mora biti prazan kad se nije moglo spojiti")
	}
}

func TestStatus(t *testing.T) {
	tests := []struct {
		name string
		res  Result
		want Status
	}{
		{"puno dana", Result{DaysLeft: 60}, OK},
		{"tocno na granici", Result{DaysLeft: 21}, OK},
		{"ispod granice", Result{DaysLeft: 20}, Warn},
		{"istekao", Result{DaysLeft: -1, Err: errors.New("expired")}, Fail},
		{"greska s puno dana", Result{DaysLeft: 60, Err: errors.New("unknown authority")}, Fail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.res.Status(21); got != tt.want {
				t.Errorf("Status(21) = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckAll_KeepsOrder(t *testing.T) {
	srv, roots := tlsServer(t)
	addr := srv.Listener.Addr().String()

	domains := []string{addr, "localhost:1", addr, addr}
	results := Checker{Roots: roots, Timeout: time.Second}.CheckAll(t.Context(), domains, 2)

	for i, r := range results {
		if r.Domain != domains[i] {
			t.Errorf("results[%d].Domain = %s, want %s", i, r.Domain, domains[i])
		}
	}
	if results[1].Err == nil {
		t.Error("localhost:1 bi trebao pasti")
	}
}
