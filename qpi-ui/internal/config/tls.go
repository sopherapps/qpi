package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"net"
	"os"
	"time"
)

const (
	certificatePemHeader = "CERTIFICATE"
	privateKeyPemHeader  = "PRIVATE KEY"
)

// cert key pair
type certKeyPair struct {
	Certificate *x509.Certificate
	PrivateKey  *rsa.PrivateKey
}

// loadTLS loads the TLS from the given TLS certificate file and key file
func loadTLS(certFile string, keyFile string) (*tls.Config, error) {
	isValidCert, err := isValidPemOfType(certFile, certificatePemHeader)
	if err != nil {
		return nil, fmt.Errorf("error validating certificate file '%s': %w", certFile, err)
	}

	isValidKey, err := isValidPemOfType(keyFile, privateKeyPemHeader)
	if err != nil {
		return nil, fmt.Errorf("error validating private key file '%s': %w", keyFile, err)
	}

	if !isValidCert || !isValidKey {
		return nil, fmt.Errorf("key '%s' or/and certificate '%s' are expired or empty", keyFile, certFile)
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("error loading TLS certificate '%s' and key '%s': %w", certFile, keyFile, err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// getTlsCertKeyPair gets the parsed TLS certificate file and key file or generates them if they are invalid
func getTlsCertKeyPair(certFile string, keyFile string, domain string, ca *certKeyPair) (*certKeyPair, error) {
	isValidCert, err := isValidPemOfType(certFile, certificatePemHeader)
	if err != nil {
		return nil, fmt.Errorf("error validating certificate file '%s': %w", certFile, err)
	}

	isValidKey, err := isValidPemOfType(keyFile, privateKeyPemHeader)
	if err != nil {
		return nil, fmt.Errorf("error validating private key file '%s': %w", keyFile, err)
	}

	if !isValidCert || !isValidKey {
		err = generateCertAndKeyFiles(domain, certFile, keyFile, ca)
		if err != nil {
			return nil, err
		}
	}

	return readCertKeyPair(certFile, keyFile)
}

// getCA gets the certificate authority for generating TLS certificates
func getCA(caCertPath, caKeyPath string) (*certKeyPair, error) {
	isValidCert, err := isValidPemOfType(caCertPath, certificatePemHeader)
	if err != nil {
		return nil, fmt.Errorf("error validating certificate file '%s': %w", caCertPath, err)
	}

	isValidKey, err := isValidPemOfType(caKeyPath, privateKeyPemHeader)
	if err != nil {
		return nil, fmt.Errorf("error validating private key file '%s': %w", caKeyPath, err)
	}

	if !isValidCert || !isValidKey {
		return generateCA(caCertPath, caKeyPath)
	}

	return readCertKeyPair(caCertPath, caKeyPath)
}

// generateCA creates certificate authority for generating TLS certificates
func generateCA(caCertPath, caKeyPath string) (*certKeyPair, error) {
	// Create a 10-year Root CA Template
	rootKey, _ := rsa.GenerateKey(rand.Reader, 4096)
	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	template := &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{Organization: []string{"Quantum Processing Interface CA"}},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 10 Years
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	// Self-sign the root certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &rootKey.PublicKey, rootKey)
	if err != nil {
		return nil, fmt.Errorf("error creating certificate: %w", err)
	}

	// Encode Private Key to modern PKCS#8 PEM format
	rootKeyBytes, err := x509.MarshalPKCS8PrivateKey(rootKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key to PKCS8: %w", err)
	}

	// Write the Certificate File
	// Permissions: 0644 (Owner can read/write; everyone else can read)
	err = writePEMToFile(caCertPath, certificatePemHeader, derBytes, 0644)
	if err != nil {
		return nil, err
	}

	// Write the Private Key File
	// CRITICAL Security Step: Permissions MUST be 0600
	// (Only the owner/process running the app can read or write this file)
	err = writePEMToFile(caKeyPath, privateKeyPemHeader, rootKeyBytes, 0600)
	if err != nil {
		return nil, err
	}

	return &certKeyPair{template, rootKey}, nil
}

// generateCertAndKeyFiles generates a certificate and key PEM files for use in TLS handshakes
// and all that and saves them to the given locations.
func generateCertAndKeyFiles(domain string, certFile string, keyFile string, ca *certKeyPair) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Setup Subject Alternative Names (SANs)
	dnsNames := []string{"localhost"}
	if domain != "" && domain != "localhost" {
		dnsNames = append(dnsNames, domain)
	}

	hostIPs, err := fetchHostIPs()
	if err != nil {
		return fmt.Errorf("failed auto-detecting host IPs: %w", err)
	}

	ipAddresses := append([]net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}, hostIPs...)

	// Build the x509 Template Constraints
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Quantum Processing Interface"},
			CommonName:   domain,
		},
		NotBefore: time.Now(),
		// 7 days
		NotAfter:              time.Now().Add(7 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
		IPAddresses:           ipAddresses,
	}

	// Sign key with the root CA
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, ca.Certificate, &privateKey.PublicKey, ca.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode Private Key to modern PKCS#8 PEM format
	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key to PKCS8: %w", err)
	}

	// Write the Certificate File
	// Permissions: 0644 (Owner can read/write; everyone else can read)
	err = writePEMToFile(certFile, certificatePemHeader, derBytes, 0644)
	if err != nil {
		return err
	}

	// Write the Private Key File
	// CRITICAL Security Step: Permissions MUST be 0600
	// (Only the owner/process running the app can read or write this file)
	err = writePEMToFile(keyFile, privateKeyPemHeader, privBytes, 0600)
	if err != nil {
		return err
	}

	return nil
}

// readCertKeyPair reads the certificate, key pair at the given paths
func readCertKeyPair(caCertPath string, caKeyPath string) (*certKeyPair, error) {
	certBytes, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("error reading file '%s': %w", caCertPath, err)
	}
	keyBytes, err := os.ReadFile(caKeyPath)
	if err != nil {
		return nil, fmt.Errorf("error readingfile '%s': %w", caKeyPath, err)
	}

	block, _ := pem.Decode(certBytes)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing certificate '%s': %w", caCertPath, err)
	}

	keyBlock, _ := pem.Decode(keyBytes)
	parsedKey, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing key '%s': %w", caKeyPath, err)
	}

	key, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key '%s' is valid PKCS#8, but not RSA key: %w", caKeyPath, err)
	}

	return &certKeyPair{cert, key}, nil
}

// writePEMToFile encodes raw DER bytes into a PEM block and streams it directly to a disk file.
func writePEMToFile(filename string, blockType string, derBytes []byte, perm os.FileMode) error {
	// Open file with specific write flags and strict production permissions
	// os.O_TRUNC ensures if the file already exists, it is overwritten cleanly
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	// Prepare the PEM block structure
	block := &pem.Block{
		Type:  blockType,
		Bytes: derBytes,
	}

	// Stream encode directly to the file
	if err := pem.Encode(file, block); err != nil {
		return fmt.Errorf("failed to write PEM data to %s: %w", filename, err)
	}

	return nil
}

// Internal helper to scrape non-loopback network interfaces for valid IPs
func fetchHostIPs() ([]net.IP, error) {
	var ips []net.IP
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addresses {
		// Check if network address is not a loopback interface
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil || ipNet.IP.To16() != nil {
				ips = append(ips, ipNet.IP)
			}
		}
	}
	return ips, nil
}

// isValidPemOfType checks if a file is a PEM file with the given header.
func isValidPemOfType(path string, pemType string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil // File simply doesn't exist
		}
		return false, fmt.Errorf("failed to read file: %w", err)
	}

	// Attempt to decode the PEM block
	// pem.Decode returns the decoded block and any remaining un-decoded bytes.
	block, _ := pem.Decode(data)
	if block == nil {
		// The file contains text/whitespace, but NO valid PEM formatting structure
		return false, nil
	}

	// Enforce specific block headers ("CERTIFICATE" vs "PRIVATE KEY")
	if block.Type != pemType {
		return false, fmt.Errorf("found valid PEM, but header was '%s' (expected '%s')", block.Type, pemType)
	}

	if isCertExpired(block) {
		return false, fmt.Errorf("found valid PEM, but it is expired")
	}

	return true, nil
}

// isCertExpired checks if a valid certificate file has expired or is not yet valid.
// It returns true if the certificate cannot be safely used chronologically.
func isCertExpired(body *pem.Block) bool {
	cert, err := x509.ParseCertificate(body.Bytes)
	if err != nil {
		return false
	}

	if time.Now().After(cert.NotAfter) {
		return true
	}

	return false
}

// isCertUpForRenewal checks if the certificate is expiring within the buffer window.
func isCertUpForRenewal(ckPair *certKeyPair, bufferWindow time.Duration) bool {
	// If current time + buffer window is after the expiration, renew it
	return time.Now().Add(bufferWindow).After(ckPair.Certificate.NotAfter)
}

// getCaCertHash takes the tls Config and returns a standard hex string.
func getCaCertHash(caConfig *certKeyPair) (string, error) {
	if caConfig == nil || caConfig.Certificate == nil {
		return "", fmt.Errorf("tls config does not contain any active certificates")
	}

	rawDerBytes := caConfig.Certificate.Raw
	hash := sha256.Sum256(rawDerBytes)
	return hex.EncodeToString(hash[:]), nil
}
