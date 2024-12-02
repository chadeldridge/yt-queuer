package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"
	"time"

	ytqueuer "github.com/chadeldridge/yt-queuer/application"
	"github.com/chadeldridge/yt-queuer/cmd"
)

const lock_file = "ytqueuer.pid"

var (
	appName    = "ytqueuer"
	regAppName = regexp.MustCompile(appName)
	help       = fmt.Sprintf(`Usage: %s [options] [help|version|start|stop]

Options:
  -h, --help     Show this help message
  -v, --version  Show the version number

Commands:
  help     Show this help message
  version  Show the version number
  start    Start the ytqueuer server
  stop     Stop the currently running ytqueuer server

Examples:
  ytqueuer start
  ytqueuer stop

`, appName)
)

func start(
	logger *log.Logger,
	ctx context.Context,
	addr string,
	port int,
	certFile string,
	keyFile string,
	pls ytqueuer.Playlists,
	db *ytqueuer.SqliteDB,
) error {
	// queue := ytqueuer.NewQueue()
	server := ytqueuer.NewHTTPServer(logger, addr, port, certFile, keyFile, pls, db)
	server.AddRoutes()

	srvErr := make(chan error)
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			srvErr <- err
		}
		close(srvErr)
	}()

	<-done

	err := server.Stop(ctx, 10)
	if err != nil {
		return fmt.Errorf("error while stopping ytqueuer: %w", err)
	}
	return <-srvErr
}

func stop(logger *log.Logger) {
	pid := findLock(logger)
	if pid == 0 {
		logger.Println("ytqueuer is not running")
		os.Exit(0)
	}

	proc, err := findProc(logger, pid)
	if err != nil {
		logger.Println("ytqueuer is not running")
		os.Exit(0)
	}

	logger.Println("shutting down ytqueuer...")
	err = proc.Signal(os.Interrupt)
	if err != nil {
		logger.Printf("error while stopping ytqueuer (%d): %v\n", pid, err)
		os.Exit(1)
	}
}

func main() {
	ctx := context.Background()
	logger := log.New(os.Stdout, "ytqueuer: ", log.LstdFlags)

	// Parse args. Right now we only look for stop. If stop is passed the program will exit.
	parseArgs(logger)

	// Verify if ytqueuer is already running. If it is, exit. If not, remove the lock file and
	// save our pid.
	lock(logger)

	addr := ""
	port := 8080
	certFile, keyFile, err := verifyCerts(logger, "certs", "certificate.crt", "privatekey.key")
	if err != nil {
		log.Printf("error: %v\n", err)
		os.Exit(1)
	}

	// Create a playlist cache.
	pls := ytqueuer.NewPlaylists()

	// Create a new sqlite3 database.
	db, err := initDB()
	if err != nil {
		log.Printf("error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Load our client and playlist data from the database.
	pls.LoadFromDB(db)

	/*
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		done := make(chan bool, 1)

		go func() {
			sig := <-sigs
			logger.Printf("received shutdown signal: %v\n", sig)
			deleteLock(logger)
			fmt.Println("ytqueuer stopped")
			done <- true
		}()
	*/

	if err := start(logger, ctx, addr, port, certFile, keyFile, pls, db); err != nil {
		log.Println(err)
		deleteLock(logger)
		os.Exit(1)
	}

	//<-done
	deleteLock(logger)
}

func parseArgs(logger *log.Logger) {
	if len(os.Args) != 2 {
		return
	}

	switch os.Args[1] {
	case "help", "-h", "--help":
		cmd.Help(logger, help)
	case "version", "-v", "--version":
		cmd.Version(logger, appName)
	case "stop":
		stop(logger)
		os.Exit(0)
	case "start":
		return
	default:
		logger.Printf("unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func lock(logger *log.Logger) {
	// Read the pid from the lock file. /var/run/ytqueuer.pid
	pid := findLock(logger)
	if pid == 0 {
		// Save our pid to the lock file.
		writeLock(logger, os.Getpid())
		return
	}

	_, err := findProc(logger, pid)
	if err == nil {
		logger.Printf("ytqueuer is already running as process %d\n", pid)
		os.Exit(1)
	}

	deleteLock(logger)
	writeLock(logger, os.Getpid())
}

// findLock reads the lock file and returns the pid of the running ytqueuer process. It returns 0
// if the lock file does not exist and exits if there is an error.
func findLock(logger *log.Logger) int {
	data, err := os.ReadFile(lock_file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0
		}

		logger.Printf("error while reading %s: %v\n", lock_file, err)
		os.Exit(1)
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		logger.Printf("ytqueuer.pid did not contain a pid int: %v\n", err)
		os.Exit(1)
	}

	return pid
}

func findProc(logger *log.Logger, pid int) (*os.Process, error) {
	proc, err := os.FindProcess(pid)
	// If the process is running, check if it is ytqueuer.
	if err != nil {
		return nil, err
	}

	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return nil, err
	}

	cmd, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		logger.Printf("cannot determine if process %d is ytqueuer: %v\n", pid, err)
		os.Exit(1)
	}

	// Exit if ytqueuer is already running.
	if regAppName.Match(cmd) {
		return proc, nil
	}

	return nil, fmt.Errorf("process %d is not ytqueuer", pid)
}

func writeLock(logger *log.Logger, pid int) {
	err := os.WriteFile(lock_file, []byte(strconv.Itoa(pid)), 0o644)
	if err != nil {
		logger.Printf("error while writing %s: %v\n", lock_file, err)
		os.Exit(1)
	}
}

func deleteLock(logger *log.Logger) {
	if err := os.Remove(lock_file); err != nil {
		logger.Printf("error while removing %s: %v\n", lock_file, err)
		os.Exit(1)
	}
}

func verifyCerts(logger *log.Logger, dir, cert, key string) (string, string, error) {
	certFile := dir + "/" + cert
	keyFile := dir + "/" + key

	if _, err := os.Stat(dir); err != nil {
		if !os.IsNotExist(err) {
			return certFile, keyFile, err
		}

		cwd, err := os.Getwd()
		if err != nil {
			return certFile, keyFile, err
		}

		logger.Printf("creating %s ...", cwd+"/"+dir)
		if err := os.Mkdir("certs", 0o755); err != nil {
			return certFile, keyFile, err
		}
		logger.Printf("done\n")

		return createCertFiles(certFile, keyFile)
	}

	if _, err := os.Stat(certFile); err == nil {
		if _, err := os.Stat(keyFile); err == nil {
			return certFile, keyFile, nil
		}
	}

	return createCertFiles(certFile, keyFile)
}

func createCertFiles(certFile, keyFile string) (string, string, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	})

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * 10 * time.Hour)

	// Taken from crypto/tls/generate_cert.go
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("Failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{Organization: []string{"ytqueuer"}},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return "", "", err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})

	err = os.WriteFile(certFile, certPEM, 0o644)
	if err != nil {
		return "", "", err
	}

	err = os.WriteFile(keyFile, keyPEM, 0o600)
	if err != nil {
		return "", "", err
	}

	return certFile, keyFile, nil
}

func initDB() (*ytqueuer.SqliteDB, error) {
	db, err := ytqueuer.NewSqliteDB("ytqueuer.db")
	if err != nil {
		return nil, err
	}

	err = db.Open()
	if err != nil {
		return nil, err
	}

	err = db.Migrate()
	if err != nil {
		return nil, err
	}

	return db, nil
}
