# tlscert

This is a simple tool to generate self-signed certificates for testing purposes.

## Motivation

This package is intended to be used in tests that require a self-signed certificate. It is not intended to be used in production code.

I many times found myself needing to generate a self-signed certificate for testing purposes, and I always had to look up how to do it. This package is an attempt to make this process easier, providing a simple API to generate self-signed certificates and save them to disk, if needed.

## Features

The package exposes the following types and functions:

### Types

- `Request`: A struct that contains the parameters for the certificate generation.
- `Certificate`: A struct that contains the generated certificate and key, including the paths to the files on disk.

### Functions

- `SelfSigned`: A function that generates a self-signed certificate and returns it as a `Certificate` value. This function only receives the host name for the certificate, and it does not return an error, if it occurs.
- `SelfSignedCA`: A function that generates a self-signed certificate for a Certificate Authority, and it does not return an error, if it occurs.
- `SelfSignedFromRequest`: A function that generates a self-signed certificate based on the parameters in a `Request` value, and it does not return an error, if it occurs.

All three functions have an `E` version that returns an error, if it occurs:

- `SelfSignedE`: A function that generates a self-signed certificate and returns it as a `Certificate` value. This function only receives the host name for the certificate, and it does return an error, if it occurs.
- `SelfSignedCAE`: A function that generates a self-signed certificate for a Certificate Authority, and it does return an error, if it occurs.
- `SelfSignedFromRequestE`: A function that generates a self-signed certificate based on the parameters in a `Request` value, and it does return an error, if it occurs.

Therefore, it's possible to issue a self-signed certificate with a custom host name, and save it to disk, if needed, or to issue a certificate based on a parent certificate, which is useful for generating client certificates.

The `Request` struct also provides a `ParentDir` option that can be used to save the generated certificate to disk as a PEM file.

The `Certificate` struct provides a `Transport` method, which returns a pointer to a `http.Transport` that can be used to perform HTTP requests using the generated certificate; and a `TLSConfig` method, which returns a pointer to a `tls.Config`. The `Transport` method internally uses the `TLSConfig` method.

## Example

You can find a simple example in the [example_test.go](example_test.go) file:

```go
package tlscert_test

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/mdelapenya/tlscert"
)

func ExampleSelfSigned() {
	tmp := os.TempDir()
	certsDir := tmp + "/certs"
	defer os.RemoveAll(certsDir)

	if err := os.MkdirAll(certsDir, 0o755); err != nil {
		log.Println(err)
		return
	}

	// Generate a certificate for localhost and save it to disk.
	caCert := tlscert.SelfSignedFromRequest(tlscert.Request{
		Host:      "localhost",
		Name:      "ca-cert",
		ParentDir: certsDir,
	})
	if caCert == nil {
		log.Println("Failed to generate CA certificate")
		return
	}

	cert := tlscert.SelfSignedFromRequest(tlscert.Request{
		Host:      "localhost",
		Name:      "client-cert",
		Parent:    caCert,
		ParentDir: certsDir,
	})
	if cert == nil {
		log.Println("Failed to generate certificate")
		return
	}

	// create an http server that uses the generated certificate
	// and private key to serve requests over HTTPS

	server := &http.Server{
		Addr: ":8443",
	}

	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		//nolint:errcheck
		w.Write([]byte("TLS works!\n"))
	})

	go func() {
		if err := server.ListenAndServeTLS(cert.CertPath, cert.KeyPath); err != nil {
			log.Printf("Failed to start server: %v", err)
		}
	}()
	defer server.Close()

	// perform an HTTP request to the server, using the generated certificate

	const url = "https://localhost:8443/hello"

	client := &http.Client{Transport: cert.Transport()}
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("Failed to get response: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		return
	}

	fmt.Println(string(body))

	// Output:
	// TLS works!
}
```

Or, if you prefer, you can use the `E` versions of the functions:

```go
func ExampleSelfSignedE() {
	tmp := os.TempDir()
	certsDir := tmp + "/certs"
	defer os.RemoveAll(certsDir)

	if err := os.MkdirAll(certsDir, 0o755); err != nil {
		log.Println(err)
		return
	}

	// Generate a certificate for localhost and save it to disk.
	caCert, err := tlscert.SelfSignedFromRequestE(tlscert.Request{
		Host:      "localhost",
		Name:      "ca-cert",
		ParentDir: certsDir,
	})
	if err != nil {
		log.Println("Failed to generate CA certificate")
		return
	}
	if caCert == nil {
		log.Println("Failed to generate CA certificate")
		return
	}

	cert, err := tlscert.SelfSignedFromRequestE(tlscert.Request{
		Host:      "localhost",
		Name:      "client-cert",
		Parent:    caCert,
		ParentDir: certsDir,
	})
	if err != nil {
		log.Println("Failed to generate certificate")
		return
	}
	if cert == nil {
		log.Println("Failed to generate certificate")
		return
	}

	// create an http server that uses the generated certificate
	// and private key to serve requests over HTTPS

	server := &http.Server{
		Addr: ":8443",
	}

	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		//nolint:errcheck
		w.Write([]byte("TLS works!\n"))
	})

	go func() {
		if err := server.ListenAndServeTLS(cert.CertPath, cert.KeyPath); err != nil {
			log.Printf("Failed to start server: %v", err)
		}
	}()
	defer server.Close()

	// perform an HTTP request to the server, using the generated certificate

	const url = "https://localhost:8443/hello"

	client := &http.Client{Transport: cert.Transport()}
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("Failed to get response: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		return
	}

	fmt.Println(string(body))

	// Output:
	// TLS works!
}
```
