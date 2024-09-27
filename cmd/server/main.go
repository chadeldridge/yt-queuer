package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"os"
	"time"

	ytqueuer "github.com/chadeldridge/yt-queuer"
)

func run(
	logger *log.Logger,
	ctx context.Context,
	addr string,
	port int,
	certFile string,
	keyFile string,
	q *ytqueuer.Queue,
) error {
	// queue := ytqueuer.NewQueue()
	server := ytqueuer.NewHTTPServer(logger, addr, port, certFile, keyFile, q)
	server.AddRoutes()

	return server.Start(ctx, 10)
}

func main() {
	ctx := context.Background()
	logger := log.New(os.Stdout, "ytqueuer: ", log.LstdFlags)

	// addr := "172.19.120.11"
	addr := ""
	port := 8080
	certFile, keyFile, err := verifyCerts("certs", "certificate.crt", "privatekey.key")
	if err != nil {
		log.Printf("error: %v\n", err)
		os.Exit(1)
	}

	// Create a new empty queue.
	q := ytqueuer.NewQueue()

	if err := run(logger, ctx, addr, port, certFile, keyFile, &q); err != nil {
		log.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

func verifyCerts(dir, cert, key string) (string, string, error) {
	certFile := dir + "/" + cert
	keyFile := dir + "/" + key

	if _, err := os.Stat(dir); err != nil {
		if !os.IsNotExist(err) {
			return certFile, keyFile, err
		}

		if err := os.Mkdir("certs", 0o755); err != nil {
			return certFile, keyFile, err
		}
	}

	if _, err := os.Stat(certFile); err == nil {
		if _, err := os.Stat(keyFile); err == nil {
			return certFile, keyFile, nil
		}
	}

	if err := createCertFiles(certFile, keyFile); err != nil {
		return certFile, keyFile, err
	}

	return certFile, keyFile, nil
}

func createCertFiles(certFile, keyFile string) error {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	})

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * 10 * time.Hour)

	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"ytqueuer"}},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})

	err = os.WriteFile(certFile, certPEM, 0o644)
	if err != nil {
		return err
	}

	err = os.WriteFile(keyFile, keyPEM, 0o600)
	if err != nil {
		return err
	}

	return nil
}
