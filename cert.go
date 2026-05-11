package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"slices"
	"sync"
	"time"
)

// global recycle buffer
var (
	certCache sync.Map
)

// cert
func GetOrCreateCert(chi *tls.ClientHelloInfo, defaultSni string) (*tls.Certificate, error) {
	serverName := chi.ServerName
	if serverName == "" {
		serverName = defaultSni
	}
	if slices.Contains(chi.SignatureSchemes, tls.Ed25519) {
		return GetOrCreateEd25519(serverName)
	}
	return GetOrCreateCertRSA(serverName)
}

func GetOrCreateEd25519(host string) (*tls.Certificate, error) {
	key := "ed25519:" + host
	if v, ok := certCache.Load(key); ok {
		return v.(*tls.Certificate), nil
	}

	// generate ed25519 keypair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		// ignore
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName: host,
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(24 * time.Hour),

		KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},

		// BasicConstraintsValid: true,

		DNSNames: []string{host},
	}

	// self-sign
	derBytes, _ := x509.CreateCertificate(rand.Reader, &template, &template, pub, priv)
	if err != nil {
		// ignore
	}

	cert := tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  priv,
	}

	certCache.Store(key, &cert)
	return &cert, nil
}

func GetOrCreateCertRSA(host string) (*tls.Certificate, error) {
	key := "rsa:" + host
	if v, ok := certCache.Load(key); ok {
		return v.(*tls.Certificate), nil
	}

	priv, _ := rsa.GenerateKey(rand.Reader, 2048)

	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName: host,
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(24 * time.Hour),

		KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},

		DNSNames: []string{host},
	}

	derBytes, _ := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)

	cert := tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  priv,
	}

	certCache.Store(key, &cert)
	return &cert, nil
}
