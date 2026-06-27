package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestIsValidPemOfType_MissingFile verifies that a missing file returns false without error.
func TestIsValidPemOfType_MissingFile(t *testing.T) {
	valid, err := isValidPemOfType("/nonexistent/path/to/file.pem", certificatePemHeader)
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if valid {
		t.Fatal("expected false for missing file")
	}
}

// TestIsValidPemOfType_InvalidContent verifies that garbage content returns false.
func TestIsValidPemOfType_InvalidContent(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "invalid_*.pem")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString("this is not a pem"); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	tmpFile.Close()

	valid, err := isValidPemOfType(tmpFile.Name(), certificatePemHeader)
	if err != nil {
		t.Fatalf("expected no error for invalid content, got: %v", err)
	}
	if valid {
		t.Fatal("expected false for invalid PEM content")
	}
}

// TestIsValidPemOfType_WrongType verifies that a valid PEM with wrong header returns error.
func TestIsValidPemOfType_WrongType(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "wrong_type_*.pem")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write a private key PEM but ask for certificate
	block := &pem.Block{Type: privateKeyPemHeader, Bytes: []byte("dummy")}
	if err := pem.Encode(tmpFile, block); err != nil {
		t.Fatalf("failed to encode: %v", err)
	}
	tmpFile.Close()

	_, err = isValidPemOfType(tmpFile.Name(), certificatePemHeader)
	if err == nil {
		t.Fatal("expected error for wrong PEM type")
	}
}

// TestGenerateCAAndReadBack verifies CA generation and reading it back.
func TestGenerateCAAndReadBack(t *testing.T) {
	tmpDir := t.TempDir()
	caCertPath := filepath.Join(tmpDir, "ca.pem")
	caKeyPath := filepath.Join(tmpDir, "ca.key")

	caPair, err := generateCA(caCertPath, caKeyPath)
	if err != nil {
		t.Fatalf("generateCA failed: %v", err)
	}

	if caPair.Certificate == nil {
		t.Fatal("expected non-nil CA certificate")
	}
	if caPair.PrivateKey == nil {
		t.Fatal("expected non-nil CA private key")
	}
	if !caPair.Certificate.IsCA {
		t.Fatal("expected CA certificate to have IsCA=true")
	}

	// Verify files exist with correct permissions
	certInfo, err := os.Stat(caCertPath)
	if err != nil {
		t.Fatalf("CA cert file not found: %v", err)
	}
	if certInfo.Mode().Perm() != 0644 {
		t.Fatalf("expected CA cert permissions 0644, got %04o", certInfo.Mode().Perm())
	}

	keyInfo, err := os.Stat(caKeyPath)
	if err != nil {
		t.Fatalf("CA key file not found: %v", err)
	}
	if keyInfo.Mode().Perm() != 0600 {
		t.Fatalf("expected CA key permissions 0600, got %04o", keyInfo.Mode().Perm())
	}

	// Read back and verify
	readPair, err := readCertKeyPair(caCertPath, caKeyPath)
	if err != nil {
		t.Fatalf("readCertKeyPair failed: %v", err)
	}
	if readPair.Certificate.SerialNumber.Cmp(caPair.Certificate.SerialNumber) != 0 {
		t.Fatal("read CA serial number mismatch")
	}

	// Test CA Hash generation from freshly generated CA ensures Raw is populated
	hash, err := getCaCertHash(caPair)
	if err != nil {
		t.Fatalf("getCaCertHash failed: %v", err)
	}
	// e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855 is SHA-256 of empty string
	if hash == "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Fatal("getCaCertHash returned the hash of an empty string, Raw bytes must be populated")
	}
}

// TestGetCA_ExistingValid verifies getCA returns existing valid CA without regenerating.
func TestGetCA_ExistingValid(t *testing.T) {
	tmpDir := t.TempDir()
	caCertPath := filepath.Join(tmpDir, "ca.pem")
	caKeyPath := filepath.Join(tmpDir, "ca.key")

	// Generate once
	firstPair, err := generateCA(caCertPath, caKeyPath)
	if err != nil {
		t.Fatalf("first generateCA failed: %v", err)
	}

	// Get again — should read existing
	secondPair, err := getCA(caCertPath, caKeyPath)
	if err != nil {
		t.Fatalf("getCA failed: %v", err)
	}

	if secondPair.Certificate.SerialNumber.Cmp(firstPair.Certificate.SerialNumber) != 0 {
		t.Fatal("getCA regenerated CA when it should have reused existing")
	}
}

// TestGetCA_GeneratesWhenMissing verifies getCA generates when files don't exist.
func TestGetCA_GeneratesWhenMissing(t *testing.T) {
	tmpDir := t.TempDir()
	caCertPath := filepath.Join(tmpDir, "ca.pem")
	caKeyPath := filepath.Join(tmpDir, "ca.key")

	caPair, err := getCA(caCertPath, caKeyPath)
	if err != nil {
		t.Fatalf("getCA failed: %v", err)
	}
	if caPair.Certificate == nil || caPair.PrivateKey == nil {
		t.Fatal("expected generated CA to have certificate and key")
	}
}

// TestGenerateCertAndKeyFiles verifies leaf certificate generation signed by CA.
func TestGenerateCertAndKeyFiles(t *testing.T) {
	tmpDir := t.TempDir()
	caCertPath := filepath.Join(tmpDir, "ca.pem")
	caKeyPath := filepath.Join(tmpDir, "ca.key")
	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")

	caPair, err := generateCA(caCertPath, caKeyPath)
	if err != nil {
		t.Fatalf("generateCA failed: %v", err)
	}

	domain := "test.example.com"
	customIp := "192.0.0.0"
	err = generateCertAndKeyFiles(domain, certPath, keyPath, caPair, customIp)
	if err != nil {
		t.Fatalf("generateCertAndKeyFiles failed: %v", err)
	}

	// Verify files exist
	if _, err := os.Stat(certPath); err != nil {
		t.Fatalf("cert file not created: %v", err)
	}
	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("key file not created: %v", err)
	}

	// Read and verify the certificate
	pair, err := readCertKeyPair(certPath, keyPath)
	if err != nil {
		t.Fatalf("readCertKeyPair failed: %v", err)
	}

	if pair.Certificate.Subject.CommonName != domain {
		t.Fatalf("expected CN=%s, got %s", domain, pair.Certificate.Subject.CommonName)
	}

	// Verify it's signed by the CA (check issuer matches subject of CA)
	if pair.Certificate.Issuer.String() != caPair.Certificate.Subject.String() {
		t.Fatalf("certificate issuer does not match CA subject: got %s, want %s",
			pair.Certificate.Issuer.String(), caPair.Certificate.Subject.String())
	}

	// Verify SANs include localhost
	hasLocalhost := false
	for _, name := range pair.Certificate.DNSNames {
		if name == "localhost" {
			hasLocalhost = true
			break
		}
	}
	if !hasLocalhost {
		t.Fatal("expected DNSNames to include localhost")
	}

	// Verify IP addresses include loopback
	hasLoopback := false
	hasCustomIp := false
	for _, ip := range pair.Certificate.IPAddresses {
		if ip.String() == "127.0.0.1" {
			hasLoopback = true
		}
		if ip.String() == customIp {
			hasCustomIp = true
		}
	}

	if !hasLoopback {
		t.Fatal("expected IPAddresses to include 127.0.0.1")
	}

	if !hasCustomIp {
		t.Fatalf("expected IPAddresses to include %s", customIp)
	}
}

// TestGetTlsCertKeyPair_GeneratesWhenMissing verifies generation when files are missing.
func TestGetTlsCertKeyPair_GeneratesWhenMissing(t *testing.T) {
	tmpDir := t.TempDir()
	caCertPath := filepath.Join(tmpDir, "ca.pem")
	caKeyPath := filepath.Join(tmpDir, "ca.key")
	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")

	caPair, err := generateCA(caCertPath, caKeyPath)
	if err != nil {
		t.Fatalf("generateCA failed: %v", err)
	}

	pair, err := getTlsCertKeyPair(certPath, keyPath, "localhost", caPair, "127.0.0.1")
	if err != nil {
		t.Fatalf("getTlsCertKeyPair failed: %v", err)
	}
	if pair.Certificate == nil || pair.PrivateKey == nil {
		t.Fatal("expected generated cert/key pair")
	}
}

// TestGetTlsCertKeyPair_ReusesExisting verifies existing valid certs are reused.
func TestGetTlsCertKeyPair_ReusesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	caCertPath := filepath.Join(tmpDir, "ca.pem")
	caKeyPath := filepath.Join(tmpDir, "ca.key")
	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")

	caPair, err := generateCA(caCertPath, caKeyPath)
	if err != nil {
		t.Fatalf("generateCA failed: %v", err)
	}

	// Generate first time
	firstPair, err := getTlsCertKeyPair(certPath, keyPath, "localhost", caPair, "127.0.0.1")
	if err != nil {
		t.Fatalf("first getTlsCertKeyPair failed: %v", err)
	}

	// Get again — should reuse
	secondPair, err := getTlsCertKeyPair(certPath, keyPath, "localhost", caPair, "127.0.0.1")
	if err != nil {
		t.Fatalf("second getTlsCertKeyPair failed: %v", err)
	}

	if secondPair.Certificate.SerialNumber.Cmp(firstPair.Certificate.SerialNumber) != 0 {
		t.Fatal("expected same certificate to be reused")
	}
}

// TestLoadTLS_ValidPair verifies loading a valid TLS config.
func TestLoadTLS_ValidPair(t *testing.T) {
	tmpDir := t.TempDir()
	caCertPath := filepath.Join(tmpDir, "ca.pem")
	caKeyPath := filepath.Join(tmpDir, "ca.key")
	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")

	caPair, err := generateCA(caCertPath, caKeyPath)
	if err != nil {
		t.Fatalf("generateCA failed: %v", err)
	}

	err = generateCertAndKeyFiles("localhost", certPath, keyPath, caPair, "127.0.0.1")
	if err != nil {
		t.Fatalf("generateCertAndKeyFiles failed: %v", err)
	}

	tlsConfig, err := loadTLS(certPath, keyPath)
	if err != nil {
		t.Fatalf("loadTLS failed: %v", err)
	}
	if len(tlsConfig.Certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(tlsConfig.Certificates))
	}
	if tlsConfig.MinVersion != 0x0303 { // tls.VersionTLS12
		t.Fatalf("expected MinVersion TLS 1.2")
	}
}

// TestLoadTLS_InvalidPair verifies loading fails when one file is missing.
func TestLoadTLS_InvalidPair(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")

	// Create only cert, not key
	if err := os.WriteFile(certPath, []byte("invalid"), 0644); err != nil {
		t.Fatalf("failed to write dummy cert: %v", err)
	}

	_, err := loadTLS(certPath, keyPath)
	if err == nil {
		t.Fatal("expected error for invalid cert/key pair")
	}
}

// TestGetCaCertHash verifies hash consistency.
func TestGetCaCertHash(t *testing.T) {
	tmpDir := t.TempDir()
	caCertPath := filepath.Join(tmpDir, "ca.pem")
	caKeyPath := filepath.Join(tmpDir, "ca.key")

	caPair, err := generateCA(caCertPath, caKeyPath)
	if err != nil {
		t.Fatalf("generateCA failed: %v", err)
	}

	hash1, err := getCaCertHash(caPair)
	if err != nil {
		t.Fatalf("getCaCertHash failed: %v", err)
	}
	if hash1 == "" {
		t.Fatal("expected non-empty hash")
	}

	// Same CA should produce same hash
	hash2, err := getCaCertHash(caPair)
	if err != nil {
		t.Fatalf("getCaCertHash second call failed: %v", err)
	}
	if hash1 != hash2 {
		t.Fatal("expected consistent hash for same certificate")
	}

	// Different CA should produce different hash (with high probability)
	caCertPath2 := filepath.Join(tmpDir, "ca2.pem")
	caKeyPath2 := filepath.Join(tmpDir, "ca2.key")
	caPair2, err := generateCA(caCertPath2, caKeyPath2)
	if err != nil {
		t.Fatalf("second generateCA failed: %v", err)
	}

	hash3, err := getCaCertHash(caPair2)
	if err != nil {
		t.Fatalf("getCaCertHash third call failed: %v", err)
	}
	// Note: Different CAs should almost always have different hashes.
	// We only log rather than fatal to avoid extremely rare random collisions.
	if hash1 == hash3 {
		t.Logf("warning: two different CAs produced the same hash (extremely unlikely collision)")
	}
}

// TestGetCaCertHash_NilInput verifies error on nil input.
func TestGetCaCertHash_NilInput(t *testing.T) {
	_, err := getCaCertHash(nil)
	if err == nil {
		t.Fatal("expected error for nil certKeyPair")
	}

	_, err = getCaCertHash(&certKeyPair{Certificate: nil, PrivateKey: nil})
	if err == nil {
		t.Fatal("expected error for nil certificate")
	}
}

// TestIsCertUpForRenewal verifies renewal logic.
func TestIsCertUpForRenewal(t *testing.T) {
	tmpDir := t.TempDir()
	caCertPath := filepath.Join(tmpDir, "ca.pem")
	caKeyPath := filepath.Join(tmpDir, "ca.key")
	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")

	caPair, err := generateCA(caCertPath, caKeyPath)
	if err != nil {
		t.Fatalf("generateCA failed: %v", err)
	}

	err = generateCertAndKeyFiles("localhost", certPath, keyPath, caPair, "127.0.0.1")
	if err != nil {
		t.Fatalf("generateCertAndKeyFiles failed: %v", err)
	}

	pair, err := readCertKeyPair(certPath, keyPath)
	if err != nil {
		t.Fatalf("readCertKeyPair failed: %v", err)
	}

	// Certificate is valid for 7 days, so it should NOT be up for renewal with 1 hour buffer
	if isCertUpForRenewal(pair, time.Hour) {
		t.Fatal("fresh certificate should not be up for renewal")
	}

	// But it SHOULD be up for renewal with 8 days buffer (past expiry)
	if !isCertUpForRenewal(pair, 8*24*time.Hour) {
		t.Fatal("certificate should be up for renewal with buffer past expiry")
	}
}

// TestWritePEMToFile verifies file creation and permissions.
func TestWritePEMToFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.pem")

	err := writePEMToFile(path, certificatePemHeader, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("writePEMToFile failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if info.Mode().Perm() != 0644 {
		t.Fatalf("expected permissions 0644, got %04o", info.Mode().Perm())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty file")
	}
}

// TestGenerateCertAndKeyFiles_DefaultDomain verifies default domain handling.
func TestGenerateCertAndKeyFiles_DefaultDomain(t *testing.T) {
	tmpDir := t.TempDir()
	caCertPath := filepath.Join(tmpDir, "ca.pem")
	caKeyPath := filepath.Join(tmpDir, "ca.key")
	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")

	caPair, err := generateCA(caCertPath, caKeyPath)
	if err != nil {
		t.Fatalf("generateCA failed: %v", err)
	}

	// Empty domain should still work and default to localhost-only SANs
	err = generateCertAndKeyFiles("", certPath, keyPath, caPair, "127.0.0.1")
	if err != nil {
		t.Fatalf("generateCertAndKeyFiles with empty domain failed: %v", err)
	}

	pair, err := readCertKeyPair(certPath, keyPath)
	if err != nil {
		t.Fatalf("readCertKeyPair failed: %v", err)
	}

	// Should still have localhost in DNSNames
	hasLocalhost := false
	for _, name := range pair.Certificate.DNSNames {
		if name == "localhost" {
			hasLocalhost = true
			break
		}
	}
	if !hasLocalhost {
		t.Fatal("expected localhost in DNSNames for empty domain fallback")
	}
}

// TestGenerateCertAndKeyFiles_ValidIPAddress verifies that a passed IP address is included in the certificate.
func TestGenerateCertAndKeyFiles_ValidIPAddress(t *testing.T) {
	tmpDir := t.TempDir()
	caCertPath := filepath.Join(tmpDir, "ca.pem")
	caKeyPath := filepath.Join(tmpDir, "ca.key")
	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")

	caPair, err := generateCA(caCertPath, caKeyPath)
	if err != nil {
		t.Fatalf("generateCA failed: %v", err)
	}

	testIP := "192.168.1.100"
	err = generateCertAndKeyFiles("localhost", certPath, keyPath, caPair, testIP)
	if err != nil {
		t.Fatalf("generateCertAndKeyFiles with valid IP failed: %v", err)
	}

	pair, err := readCertKeyPair(certPath, keyPath)
	if err != nil {
		t.Fatalf("readCertKeyPair failed: %v", err)
	}

	hasTestIP := false
	for _, ip := range pair.Certificate.IPAddresses {
		if ip.String() == testIP {
			hasTestIP = true
			break
		}
	}

	if !hasTestIP {
		t.Fatalf("expected IP address %s to be in certificate's IPAddresses, got %v", testIP, pair.Certificate.IPAddresses)
	}
}

// TestGenerateCertAndKeyFiles_LocalhostDomain verifies localhost domain doesn't duplicate.
func TestGenerateCertAndKeyFiles_LocalhostDomain(t *testing.T) {
	tmpDir := t.TempDir()
	caCertPath := filepath.Join(tmpDir, "ca.pem")
	caKeyPath := filepath.Join(tmpDir, "ca.key")
	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")

	caPair, err := generateCA(caCertPath, caKeyPath)
	if err != nil {
		t.Fatalf("generateCA failed: %v", err)
	}

	err = generateCertAndKeyFiles("localhost", certPath, keyPath, caPair, "127.0.0.1")
	if err != nil {
		t.Fatalf("generateCertAndKeyFiles failed: %v", err)
	}

	pair, err := readCertKeyPair(certPath, keyPath)
	if err != nil {
		t.Fatalf("readCertKeyPair failed: %v", err)
	}

	// Should have localhost only once
	localhostCount := 0
	for _, name := range pair.Certificate.DNSNames {
		if name == "localhost" {
			localhostCount++
		}
	}
	if localhostCount != 1 {
		t.Fatalf("expected localhost exactly once in DNSNames, got %d", localhostCount)
	}
}

// TestIsCertExpired verifies expired certificate detection.
func TestIsCertExpired(t *testing.T) {
	// We test the isCertExpired function indirectly via isValidPemOfType
	// by creating a real expired cert
	tmpDir := t.TempDir()
	caCertPath := filepath.Join(tmpDir, "ca.pem")
	caKeyPath := filepath.Join(tmpDir, "ca.key")
	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")

	caPair, err := generateCA(caCertPath, caKeyPath)
	if err != nil {
		t.Fatalf("generateCA failed: %v", err)
	}

	// Generate a cert with very short expiry
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	shortTemplate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Test"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now().Add(-2 * time.Hour),
		NotAfter:              time.Now().Add(-1 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &shortTemplate, caPair.Certificate, &privateKey.PublicKey, caPair.PrivateKey)
	if err != nil {
		t.Fatalf("failed to create expired cert: %v", err)
	}

	privBytes, _ := x509.MarshalPKCS8PrivateKey(privateKey)
	writePEMToFile(certPath, certificatePemHeader, derBytes, 0644)
	writePEMToFile(keyPath, privateKeyPemHeader, privBytes, 0600)

	valid, err := isValidPemOfType(certPath, certificatePemHeader)
	if err == nil && valid {
		t.Fatal("expected expired certificate to be invalid")
	}
}
