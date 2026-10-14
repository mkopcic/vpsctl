// Package certs provjerava TLS certifikate domena — istek, lanac i hostname.
// Ne treba SSH: spaja se na domenu kao browser, na port 443.
package certs

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"math"
	"net"
	"strings"
	"sync"
	"time"
)

type Status int

const (
	OK Status = iota
	Warn
	Fail
)

// Result je ishod provjere jedne domene.
// NotAfter i DaysLeft su popunjeni i kad je cert nevaljan (npr. istekao),
// samo ne kad se na domenu uopce nije moglo spojiti.
type Result struct {
	Domain   string
	NotAfter time.Time
	DaysLeft int
	Issuer   string
	Err      error
}

// Status klasificira rezultat: Fail ako ima greske, Warn ako istice za manje od warnDays
func (r Result) Status(warnDays int) Status {
	switch {
	case r.Err != nil:
		return Fail
	case r.DaysLeft < warnDays:
		return Warn
	default:
		return OK
	}
}

// Checker drzi postavke provjere. Nulta vrijednost radi s razumnim defaultima.
type Checker struct {
	Timeout time.Duration    // 0 = 5s
	Roots   *x509.CertPool   // nil = sistemski CA (kao browser)
	Now     func() time.Time // nil = time.Now; testovi podmecu svoje vrijeme
}

func (c Checker) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

// Check spaja se na domenu i provjerava certifikat.
// domain moze biti "mellon.hr" ili "mellon.hr:8443".
func (c Checker) Check(ctx context.Context, domain string) Result {
	res := Result{Domain: domain}

	host, port, err := net.SplitHostPort(domain)
	if err != nil {
		host, port = domain, "443"
	}

	timeout := c.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	d := tls.Dialer{
		NetDialer: &net.Dialer{Timeout: timeout},
		Config: &tls.Config{
			ServerName: host,
			// Verifikaciju radimo sami u verify(). Da je ostavimo TLS-u, handshake
			// bi samo pao i ne bismo saznali datum isteka nevaljanog certa.
			InsecureSkipVerify: true, // #nosec G402 -- rucna verifikacija ispod
		},
	}

	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(host, port))
	if err != nil {
		res.Err = err
		return res
	}
	defer conn.Close()

	chain := conn.(*tls.Conn).ConnectionState().PeerCertificates
	if len(chain) == 0 {
		res.Err = errors.New("server nije poslao certifikat")
		return res
	}

	leaf := chain[0]
	now := c.now()

	res.NotAfter = leaf.NotAfter
	res.Issuer = issuerName(leaf)
	// Floor, ne odsijecanje — cert koji je istekao prije 3 sata ima -1 dan, ne 0
	res.DaysLeft = int(math.Floor(leaf.NotAfter.Sub(now).Hours() / 24))
	res.Err = c.verify(chain, host, now)
	if res.Err == nil && incompleteChain(chain) {
		res.Err = ErrIncompleteChain
	}

	return res
}

// ErrIncompleteChain: server salje samo svoj cert, bez intermediatea.
// Browser i Windows to sami poprave (dohvate intermediate preko AIA),
// ali curl, PHP/Guzzle i Go na Linuxu odbiju konekciju.
var ErrIncompleteChain = errors.New("server ne salje intermediate cert")

// incompleteChain: lanac ima samo leaf, a leaf nije self-signed.
// Javni CA-ovi nikad ne potpisuju leaf direktno rootom, pa leaf bez
// ijednog intermediatea znaci da je server krivo konfiguriran.
func incompleteChain(chain []*x509.Certificate) bool {
	leaf := chain[0]
	return len(chain) == 1 && !bytes.Equal(leaf.RawIssuer, leaf.RawSubject)
}

// verify provjerava lanac do root CA, hostname i datume — isto sto radi browser
func (c Checker) verify(chain []*x509.Certificate, host string, now time.Time) error {
	intermediates := x509.NewCertPool()
	for _, cert := range chain[1:] {
		intermediates.AddCert(cert)
	}

	_, err := chain[0].Verify(x509.VerifyOptions{
		DNSName:       host,
		Roots:         c.Roots,
		Intermediates: intermediates,
		CurrentTime:   now,
	})
	return err
}

func issuerName(cert *x509.Certificate) string {
	if cert.Issuer.CommonName != "" {
		return cert.Issuer.CommonName
	}
	return strings.Join(cert.Issuer.Organization, ", ")
}

// CheckAll provjerava sve domene paralelno, najvise `parallel` odjednom.
// Rezultati su u istom redoslijedu kao domains.
func (c Checker) CheckAll(ctx context.Context, domains []string, parallel int) []Result {
	results := make([]Result, len(domains))

	// Semafor: buffered channel kapaciteta `parallel` — goroutina mora
	// ubaciti token prije rada, pa ih nikad ne radi vise od `parallel`
	sem := make(chan struct{}, parallel)

	var wg sync.WaitGroup
	for i, domain := range domains {
		// wg.Go (Go 1.25) = wg.Add(1) + go func + defer wg.Done()
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()

			// Svaka goroutina pise u svoj indeks — nema dijeljenja, ne treba mutex
			results[i] = c.Check(ctx, domain)
		})
	}
	wg.Wait()

	return results
}

// Explain prevodi tipicne x509 greske u jasnu poruku
func Explain(err error) string {
	if err == nil {
		return ""
	}

	if errors.Is(err, ErrIncompleteChain) {
		return "server ne salje intermediate cert — browser radi, ali curl/PHP/API klijenti padaju"
	}
	if invalid, ok := errors.AsType[x509.CertificateInvalidError](err); ok && invalid.Reason == x509.Expired {
		return "certifikat je istekao (ili jos nije valjan)"
	}
	if hostErr, ok := errors.AsType[x509.HostnameError](err); ok {
		valid := hostErr.Certificate.DNSNames
		if len(valid) > 4 {
			valid = append(valid[:4:4], "...")
		}
		return fmt.Sprintf("certifikat ne pokriva %s (vrijedi za: %s)", hostErr.Host, strings.Join(valid, ", "))
	}
	if _, ok := errors.AsType[x509.UnknownAuthorityError](err); ok {
		return "nepoznat izdavatelj — self-signed ili server ne salje intermediate cert"
	}
	if netErr, ok := errors.AsType[net.Error](err); ok && netErr.Timeout() {
		return "timeout — domena ne odgovara na 443"
	}
	if _, ok := errors.AsType[*net.DNSError](err); ok {
		return "domena se ne resolva (DNS)"
	}

	return err.Error()
}
