package utils

import (
	"log"
	"os"
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"google.golang.org/grpc/credentials"
)

func LoadCredentialsAsServer(caCrtPath string, crtPath string, keyPath string)(credentials.TransportCredentials){
	cert, err := tls.LoadX509KeyPair(crtPath, keyPath)
	if err != nil {
		log.Fatalf("Failed to load server certificates: %v", err)
	}

	caCert, err := os.ReadFile(caCrtPath)
	if err != nil {
		log.Fatalf("Failed to read CA certificate: %v", err)
	}

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caCert)

	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    certPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	})
	return creds
}

func LoadCredentialsAsClient(caPath string, crtPath string, keyPath string)(credentials.TransportCredentials){
	cert, err := tls.LoadX509KeyPair(crtPath, keyPath)
	if err != nil {
		log.Fatalf("Failed to load client certificates: %v", err)
	}

	caCert, err := os.ReadFile(caPath)
	if err != nil {
		log.Fatalf("Failed to read CA certificate: %v", err)
	}

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caCert)

	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      certPool,
		ServerName: "localhost",
	})
	return creds
}

var keyLogWriter *os.File

func init() {
	sslKeyLogFile := os.Getenv("SSLKEYLOGFILE")
	if sslKeyLogFile != "" {
		var err error
		keyLogWriter, err = os.OpenFile(sslKeyLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			panic(fmt.Sprintf("Failed to open SSLKEYLOGFILE: %v", err))
		}
	}
}

func LoadClientTLSConfigWithKeyLog(caFile, certFile, keyFile string) *tls.Config {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		panic(err)
	}

	caCert, err := os.ReadFile(caFile)
	if err != nil {
		panic(err)
	}

	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caCert)

	tlsConf := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caPool,
		InsecureSkipVerify: false,
	}

	if keyLogWriter != nil {
		tlsConf.KeyLogWriter = keyLogWriter
	}

	return tlsConf
}

func LoadServerTLSConfigWithKeyLog(caFile, certFile, keyFile string) *tls.Config {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		panic(err)
	}

	caCert, err := os.ReadFile(caFile)
	if err != nil {
		panic(err)
	}

	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caCert)

	tlsConf := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}

	if keyLogWriter != nil {
		tlsConf.KeyLogWriter = keyLogWriter
	}

	return tlsConf
}
