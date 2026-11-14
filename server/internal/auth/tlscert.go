package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

const certValidity = 10 * 365 * 24 * time.Hour

func EnsureServerCert(dataDir string) (certFile, keyFile, fingerprint string, err error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return "", "", "", fmt.Errorf("creating data directory: %w", err)
	}

	certFile = filepath.Join(dataDir, "server.crt")
	keyFile = filepath.Join(dataDir, "server.key")

	certExists := fileExists(certFile)
	keyExists := fileExists(keyFile)

	if certExists && keyExists {
		fingerprint, err = fingerprintCertFile(certFile)
		if err != nil {
			return "", "", "", err
		}
		return certFile, keyFile, fingerprint, nil
	}

	if certExists != keyExists {
		return "", "", "", errors.New("corrupt TLS state: certificate and key must both exist or neither")
	}

	if err := generateServerCert(certFile, keyFile); err != nil {
		return "", "", "", err
	}

	fingerprint, err = fingerprintCertFile(certFile)
	if err != nil {
		return "", "", "", err
	}

	return certFile, keyFile, fingerprint, nil
}

func generateServerCert(certFile, keyFile string) error {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generating TLS key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("generating certificate serial: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "pi-alarm-clock",
		},
		NotBefore:   time.Now().Add(-24 * time.Hour),
		NotAfter:    time.Now().Add(certValidity),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames: []string{
			"pi-alarm-clock.local",
			"localhost",
		},
		IPAddresses: append(
			localIPv4s(),
			net.ParseIP("127.0.0.1"),
			net.ParseIP("::1"),
		),
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return fmt.Errorf("creating TLS certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := writeFileAtomic(certFile, certPEM, 0o644); err != nil {
		return fmt.Errorf("writing TLS certificate: %w", err)
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return fmt.Errorf("marshalling TLS key: %w", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	if err := writeFileAtomic(keyFile, keyPEM, 0o600); err != nil {
		return fmt.Errorf("writing TLS key: %w", err)
	}

	return nil
}

func fingerprintCertFile(certFile string) (string, error) {
	pemBytes, err := os.ReadFile(certFile)
	if err != nil {
		return "", fmt.Errorf("reading TLS certificate: %w", err)
	}

	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "CERTIFICATE" {
		return "", errors.New("invalid certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parsing TLS certificate: %w", err)
	}

	if time.Now().After(cert.NotAfter) {
		return "", fmt.Errorf("TLS certificate expired at %s (delete %s to regenerate; clients will need to re-verify the new fingerprint)", cert.NotAfter, certFile)
	}

	sum := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(sum[:]), nil
}

func localIPv4s() []net.IP {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}

	var ips []net.IP

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}

		if ip := ipNet.IP.To4(); ip != nil {
			ips = append(ips, ip)
		}
	}

	return ips
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}

	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmp.Name(), path)
}
