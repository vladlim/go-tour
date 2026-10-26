package testcert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"time"
)

// New creates a localhost certificate and a root pool that trusts exactly it.
func New() (tls.Certificate, *x509.CertPool, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048); if err != nil { return tls.Certificate{}, nil, err }
	tpl := x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "networks-demo"},
		NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.IPv4(127,0,0,1)}, DNSNames: []string{"localhost"},
		BasicConstraintsValid: true, IsCA: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tpl, &tpl, &key.PublicKey, key); if err != nil { return tls.Certificate{}, nil, err }
	cert := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
	pool := x509.NewCertPool(); parsed, err := x509.ParseCertificate(der); if err != nil { return tls.Certificate{}, nil, err }; pool.AddCert(parsed)
	return cert, pool, nil
}
