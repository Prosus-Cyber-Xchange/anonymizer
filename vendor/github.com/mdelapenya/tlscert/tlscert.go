package tlscert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultValidFor   time.Duration = 365 * 24 * time.Hour
	defaultRSAKeySize               = 2048
	defaultFileMode                 = 0o644
	certificateType                 = "CERTIFICATE"
	privateKeyType                  = "RSA PRIVATE KEY"
)

// Certificate represents a certificate and private key pair. It's a wrapper
// around the x509.Certificate and rsa.PrivateKey types, and includes the raw
// bytes of the certificate and private key.
type Certificate struct {
	Cert      *x509.Certificate
	Bytes     []byte
	Key       *rsa.PrivateKey
	KeyBytes  []byte
	CertPath  string
	KeyPath   string
	tlsConfig *tls.Config
}

// TLSConfig returns a tls.Config that uses the certificate as the root CA,
// and the certificate for the server's certificate. This method will cache
// the tls.Config for future calls.
func (c *Certificate) TLSConfig() *tls.Config {
	if c.tlsConfig != nil {
		return c.tlsConfig
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(c.Cert)

	tlsCert, err := tls.X509KeyPair(c.Bytes, c.KeyBytes)
	if err != nil {
		return nil
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		RootCAs:      caCertPool,
	}

	c.tlsConfig = cfg

	return cfg
}

// Transport returns an http.Transport that uses the certificate as the root CA.
func (c *Certificate) Transport() *http.Transport {
	tlsConfig := c.TLSConfig()

	return &http.Transport{
		TLSClientConfig: tlsConfig,
	}
}

// Request represents a request to generate a self-signed X.509 certificate.
type Request struct {
	// Name of the certificate. Will be used to save the certificate and private key to disk (e.g. cert-<Name>.pem, key-<Name>.pem)
	Name string

	// CommonName is the subject name of the certificate
	SubjectCommonName string

	// Host sets the hostnames and IPs to generate a certificate for.
	// In the case the passed string contains comma-separated values,
	// it will be split into multiple hostnames and IPs. Each hostname and IP
	// will be trimmed of whitespace, and if the value is an IP, it will be
	// added to the IPAddresses field of the certificate, after the ones
	// passed with the WithIPAddresses option. Otherwise, it will be added
	// to the DNSNames field.
	Host string

	// Duration that certificate is valid for
	ValidFor time.Duration

	// IsCA sets the certificate as a Certificate Authority.
	// When passed, the KeyUsage field of the certificate
	// will append the x509.KeyUsageCertSign usage.
	IsCA bool

	// IPAddresses IP addresses to include in the Subject Alternative Name
	IPAddresses []net.IP

	// Parent the parent certificate and private key of the certificate.
	// It's used to sign the certificate with the parent certificate.
	// At the moment the parent is set, the issuer of the certificate will be
	// set to the common name of the parent certificate.
	Parent *Certificate

	// ParentDir sets the directory to save the certificate and private key.
	ParentDir string
}

// NewRequest returns a new CertRequest with default values to avoid nil pointers.
// The name of the certificate will be set to the host, replacing all commas with underscores.
// The certificate will be valid for 1 year.
func NewRequest(host string) Request {
	return Request{
		Name:        strings.ReplaceAll(host, ",", "_"),
		Host:        host,
		ValidFor:    defaultValidFor,
		IPAddresses: make([]net.IP, 0),
	}
}

// SelfSigned Generate a self-signed X.509 certificate for a TLS server. Returns
// a struct containing the certificate and private key, as well as the raw bytes
// for both of them. The raw bytes will be PEM-encoded.
// Considerations for the generated certificate are as follows:
//   - will be valid for the duration set in the ValidFor option, starting from 1 minute ago. Else, it will be valid for 1 year.
//   - will be signed by the parent certificate if the WithParent option is set. Else, it will be self-signed.
//   - will be saved to the directory set in the WithSaveToFile option. Else, it will not be saved to disk.
//   - will be its own Certificate Authority if the AsCA option is set. Else, it will not be a CA.
func SelfSigned(host string) *Certificate {
	cert, err := SelfSignedE(host)
	if err != nil {
		return nil
	}

	return cert
}

// SelfSignedE Generate a self-signed X.509 certificate for a TLS server. Returns
// a struct containing the certificate and private key, as well as the raw bytes
// for both of them, and an error. The raw bytes will be PEM-encoded.
// Considerations for the generated certificate are as follows:
//   - will be valid for the duration set in the ValidFor option, starting from 1 minute ago. Else, it will be valid for 1 year.
//   - will be signed by the parent certificate if the WithParent option is set. Else, it will be self-signed.
//   - will be saved to the directory set in the WithSaveToFile option. Else, it will not be saved to disk.
//   - will be its own Certificate Authority if the AsCA option is set. Else, it will not be a CA.
func SelfSignedE(host string) (*Certificate, error) {
	req := NewRequest(host)

	return SelfSignedFromRequestE(req)
}

// SelfSignedCA Generate a self-signed X.509 certificate for a Certificate Authority.
// This function is a wrapper around SelfSigned, with the IsCA option set to true.
func SelfSignedCA(host string) *Certificate {
	cert, err := SelfSignedCAE(host)
	if err != nil {
		return nil
	}

	return cert
}

// SelfSignedCAE Generate a self-signed X.509 certificate for a Certificate Authority,
// and an error. This function is a wrapper around SelfSignedFromRequest, with the
// IsCA option set to true.
func SelfSignedCAE(host string) (*Certificate, error) {
	req := NewRequest(host)

	req.IsCA = true

	return SelfSignedFromRequestE(req)
}

// SelfSignedFromRequest Generate a self-signed X.509 certificate for a TLS server,
// using the provided CertRequest.
func SelfSignedFromRequest(req Request) *Certificate {
	cert, err := SelfSignedFromRequestE(req)
	if err != nil {
		return nil
	}

	return cert
}

// SelfSignedFromRequestE Generate a self-signed X.509 certificate for a TLS server,
// using the provided CertRequest. Returns an error if the certificate cannot be generated.
func SelfSignedFromRequestE(req Request) (*Certificate, error) {
	var certificate *Certificate

	if len(req.Host) == 0 {
		return nil, ErrHostRequired
	}

	if req.ValidFor == 0 {
		req.ValidFor = defaultValidFor
	}

	keyUsage := x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
	if req.IsCA {
		keyUsage |= x509.KeyUsageCertSign
	}

	// certificate is not valid before 1 minute ago
	notBefore := time.Now().Add(-time.Minute)
	if req.Parent != nil {
		notBefore = req.Parent.Cert.NotBefore
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			CommonName: req.SubjectCommonName,
		},
		NotBefore:             notBefore,
		NotAfter:              notBefore.Add(req.ValidFor),
		KeyUsage:              keyUsage,
		BasicConstraintsValid: true,
		IsCA:                  req.IsCA,
	}

	if len(req.IPAddresses) > 0 {
		template.IPAddresses = req.IPAddresses
	}

	hosts := strings.Split(req.Host, ",")
	for _, h := range hosts {
		h = strings.TrimSpace(h)
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	pk, err := rsa.GenerateKey(rand.Reader, defaultRSAKeySize)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	if req.Parent == nil {
		req.Parent = &Certificate{}
		// if no parent is provided, use the generated certificate as the parent
		req.Parent.Cert = &template
		// use the generated private key
		req.Parent.Key = pk
	} else {
		// if a parent is provided, use the parent's common name as the issuer
		template.Issuer.CommonName = req.Parent.Cert.Subject.CommonName
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, req.Parent.Cert, pk.Public(), req.Parent.Key)
	if err != nil {
		return nil, fmt.Errorf("create x509 certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, fmt.Errorf("parse x509 certificate: %w", err)
	}

	certificate = &Certificate{
		Cert: cert,
		Key:  pk,
	}

	certificate.Bytes = pem.EncodeToMemory(&pem.Block{
		Type:  certificateType,
		Bytes: certBytes,
	})
	certificate.KeyBytes = pem.EncodeToMemory(&pem.Block{
		Type:  privateKeyType,
		Bytes: x509.MarshalPKCS1PrivateKey(pk),
	})

	if req.ParentDir != "" {
		id := sanitiseName(req.Name)
		certPath := filepath.Join(req.ParentDir, "cert-"+id+".pem")

		if err := os.WriteFile(certPath, certificate.Bytes, defaultFileMode); err != nil {
			return nil, fmt.Errorf("write certificate to file: %w", err)
		}
		certificate.CertPath = certPath

		if certificate.KeyBytes != nil {
			keyPath := filepath.Join(req.ParentDir, "key-"+id+".pem")
			if err := os.WriteFile(keyPath, certificate.KeyBytes, defaultFileMode); err != nil {
				return nil, fmt.Errorf("write key to file: %w", err)
			}
			certificate.KeyPath = keyPath
		}
	}

	return certificate, nil
}

// sanitiseName returns a sanitised version of the name, replacing spaces with underscores.
func sanitiseName(name string) string {
	if name == "" {
		name = time.Now().Format("2006-01-02T15:04:05")
	}

	transformers := []func(string) string{
		strings.TrimSpace,
		func(s string) string {
			return strings.ReplaceAll(s, " ", "")
		},
		func(s string) string {
			return strings.ReplaceAll(s, ":", "")
		},
		func(s string) string {
			return strings.ReplaceAll(s, "-", "")
		},
	}

	for _, t := range transformers {
		name = t(name)
	}

	return name
}
