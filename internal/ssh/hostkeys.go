package sshclient

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// UnknownHostError: host jos nije u known_hosts.
// Rjesenje je `vpsctl trust <server>` — nikad automatsko prihvacanje.
type UnknownHostError struct {
	Addr        string
	Fingerprint string
}

func (e *UnknownHostError) Error() string {
	return fmt.Sprintf("nepoznat host %s (%s) — pokreni: vpsctl trust <server>", e.Addr, e.Fingerprint)
}

// HostKeyChangedError: host je u known_hosts, ali server sada salje drugi kljuc.
// Ili je server reinstaliran, ili netko sjedi izmedju (MITM).
type HostKeyChangedError struct {
	Addr        string
	Fingerprint string
	File        string
	Line        int
}

func (e *HostKeyChangedError) Error() string {
	return fmt.Sprintf("HOST KLJUC SE PROMIJENIO za %s (sada %s) — moguci MITM! "+
		"Ako je server reinstaliran, obrisi liniju %d u %s pa pokreni vpsctl trust",
		e.Addr, e.Fingerprint, e.Line, e.File)
}

// KnownHostsPath vraca put do known_hosts fajla.
// VPSCTL_KNOWN_HOSTS ima prednost, inace ~/.ssh/known_hosts — isti fajl koji
// koristi OpenSSH, pa server kojem si vec vjerovao preko `ssh` radi odmah.
func KnownHostsPath() (string, error) {
	if p := os.Getenv("VPSCTL_KNOWN_HOSTS"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ssh", "known_hosts"), nil
}

// knownHosts omata knownhosts paket: raw je originalni callback,
// a nase metode iz njega izvlace jasnije greske i listu algoritama.
type knownHosts struct {
	raw ssh.HostKeyCallback
}

func loadKnownHosts() (*knownHosts, error) {
	path, err := KnownHostsPath()
	if err != nil {
		return nil, err
	}
	if err := ensureFile(path); err != nil {
		return nil, err
	}

	raw, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("known_hosts %s: %w", path, err)
	}
	return &knownHosts{raw: raw}, nil
}

// check provjerava kljuc i vraca nil, *UnknownHostError ili *HostKeyChangedError
func (k *knownHosts) check(hostname string, remote net.Addr, key ssh.PublicKey) error {
	err := k.raw(hostname, remote, key)

	keyErr, ok := errors.AsType[*knownhosts.KeyError](err)
	if !ok {
		return err
	}

	fp := ssh.FingerprintSHA256(key)
	if len(keyErr.Want) == 0 {
		return &UnknownHostError{Addr: hostname, Fingerprint: fp}
	}
	w := keyErr.Want[0]
	return &HostKeyChangedError{Addr: hostname, Fingerprint: fp, File: w.Filename, Line: w.Line}
}

// algorithms vraca algoritme kljuceva koje vec imamo za taj host.
//
// Bez ovoga server moze ponuditi npr. ECDSA kljuc, a mi imamo spremljen
// ed25519 — knownhosts bi to prijavio kao "kljuc promijenjen" iako nije.
// Trik: pozovemo raw callback s laznim kljucem, a KeyError.Want sadrzi
// sve kljuceve koje known_hosts ima za taj host.
func (k *knownHosts) algorithms(addr string, remote net.Addr) []string {
	fake, err := ssh.NewPublicKey(ed25519.PublicKey(make([]byte, ed25519.PublicKeySize)))
	if err != nil {
		return nil
	}

	keyErr, ok := errors.AsType[*knownhosts.KeyError](k.raw(addr, remote, fake))
	if !ok {
		return nil
	}

	seen := map[string]bool{}
	var algos []string
	add := func(a string) {
		if !seen[a] {
			seen[a] = true
			algos = append(algos, a)
		}
	}

	for _, w := range keyErr.Want {
		switch t := w.Key.Type(); t {
		case ssh.KeyAlgoRSA:
			// RSA kljuc se potpisuje s vise algoritama — sha2 prvo, SHA1 samo kao zadnja opcija
			add(ssh.KeyAlgoRSASHA512)
			add(ssh.KeyAlgoRSASHA256)
			add(ssh.KeyAlgoRSA)
		default:
			add(t)
		}
	}
	return algos
}

// HostKey je rezultat FetchHostKey — kljuc koji server nudi i kako
// se on odnosi prema known_hosts.
type HostKey struct {
	Addr        string
	Key         ssh.PublicKey
	Fingerprint string
	// Status je nil ako je kljuc vec poznat, inace *UnknownHostError ili *HostKeyChangedError
	Status error
}

var errGotKey = errors.New("host kljuc dohvacen")

// FetchHostKey spaja se na server samo do razmjene kljuceva i vraca host kljuc.
// Auth se ne radi — handshake prekidamo cim vidimo kljuc.
func FetchHostKey(ctx context.Context, cfg Config) (*HostKey, error) {
	kh, err := loadKnownHosts()
	if err != nil {
		return nil, err
	}

	var hk *HostKey
	clientConfig := &ssh.ClientConfig{
		User: cfg.User,
		// Moderni algoritmi prvo — ed25519 je ono sto zelimo spremiti
		HostKeyAlgorithms: []string{
			ssh.KeyAlgoED25519,
			ssh.KeyAlgoECDSA256, ssh.KeyAlgoECDSA384, ssh.KeyAlgoECDSA521,
			ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSASHA256,
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			hk = &HostKey{
				Addr:        hostname,
				Key:         key,
				Fingerprint: ssh.FingerprintSHA256(key),
				Status:      kh.check(hostname, remote, key),
			}
			return errGotKey
		},
	}

	// nil umjesto kh — ne zelimo da dial suzi algoritme na vec poznate
	conn, err := dial(ctx, cfg.addr(), clientConfig, nil, nil)
	if conn != nil {
		conn.Close()
	}
	if hk != nil {
		return hk, nil
	}
	return nil, err
}

// AddKnownHost dodaje liniju u known_hosts.
// Pazi da ne zalijepi novu liniju na zadnju ako fajl ne zavrsava s \n.
func AddKnownHost(addr string, key ssh.PublicKey) error {
	path, err := KnownHostsPath()
	if err != nil {
		return err
	}
	if err := ensureFile(path); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	prefix := ""
	if info, err := f.Stat(); err == nil && info.Size() > 0 {
		last := make([]byte, 1)
		if _, err := f.ReadAt(last, info.Size()-1); err != nil && err != io.EOF {
			return err
		}
		if last[0] != '\n' {
			prefix = "\n"
		}
	}

	_, err = fmt.Fprintf(f, "%s%s\n", prefix, knownhosts.Line([]string{addr}, key))
	return err
}

// ensureFile kreira known_hosts (i .ssh direktorij) ako ne postoji —
// knownhosts.New pada na nepostojecem fajlu.
func ensureFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDONLY, 0o600)
	if err != nil {
		return err
	}
	return f.Close()
}
