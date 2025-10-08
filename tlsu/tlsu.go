package tlsu

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	TlsPrivateKeyFileEnvVar = "TLS_PRIVATE_KEY_FILE"
	TlsCertFileEnvVar       = "TLS_CERT_FILE"
)

func MustResolveOrCreateTLSFiles() (string, string) {
	k, c, err := ResolveOrCreateTLSFiles()
	if err != nil {
		panic(err)
	}
	return k, c
}

func ResolveOrCreateTLSFiles() (string, string, error) {
	keyFile := os.Getenv(TlsPrivateKeyFileEnvVar)
	certFile := os.Getenv(TlsCertFileEnvVar)
	if keyFile != "" && certFile != "" {
		return keyFile, certFile, nil
	}
	return GenerateSelfSignedCert("localhost", "127.0.0.1", "::1")
}

// GenerateSelfSignedCert creates a self-signed certificate for development
// Returns the paths to the generated temporary cert and key files
// These files will be automatically deleted when the process exits
func GenerateSelfSignedCert(hosts ...string) (certFile, keyFile string, err error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour) // Valid for 1 year
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate serial number: %w", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Development"},
			CommonName:   "localhost",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Add hosts (DNS names and IP addresses)
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	// Create the certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return "", "", fmt.Errorf("failed to create certificate: %w", err)
	}

	// Create temporary certificate file
	certOut, err := os.CreateTemp("", "cert-*.pem")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp cert file: %w", err)
	}
	certFile = certOut.Name()

	// Write certificate
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		certOut.Close()
		os.Remove(certFile)
		return "", "", fmt.Errorf("failed to write cert: %w", err)
	}
	certOut.Close()

	// Create temporary key file
	keyOut, err := os.CreateTemp("", "key-*.pem")
	if err != nil {
		os.Remove(certFile)
		return "", "", fmt.Errorf("failed to create temp key file: %w", err)
	}
	keyFile = keyOut.Name()

	// Write private key
	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		keyOut.Close()
		os.Remove(certFile)
		os.Remove(keyFile)
		return "", "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes}); err != nil {
		keyOut.Close()
		os.Remove(certFile)
		os.Remove(keyFile)
		return "", "", fmt.Errorf("failed to write key: %w", err)
	}
	keyOut.Close()

	// delete on process exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		os.Remove(certFile)
		os.Remove(keyFile)
	}()

	return certFile, keyFile, nil
}
