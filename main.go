package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"log"
	"math/big"

	"github.com/quic-go/quic-go"
)

func main() {
	listener := assert(quic.ListenAddr("0.0.0.0:5000", generateTLSConfig(), nil))
	conn := assert(listener.Accept(context.Background()))
	assert(conn.AcceptStream(context.Background()))
}

func generateTLSConfig() *tls.Config {
	key := assert(rsa.GenerateKey(rand.Reader, 1024))
	template := x509.Certificate{SerialNumber: big.NewInt(2)}
	certDer := assert(x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key))
	keyPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDer})
	tlsCert := assert(tls.X509KeyPair(certPem, keyPem))
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic-ftcs-server"},
	}
}

func assert[T any](res T, err error) T {
	if err != nil {
		log.Fatalln(err)
	}
	return res
}
