// Command synthetic-csr verifies a bounded PKCS#10 request for the synthetic Nest harness.
package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"os"
)

func main() {
	raw, err := io.ReadAll(io.LimitReader(os.Stdin, 16385))
	block, rest := pem.Decode(raw)
	if err != nil || len(raw) > 16384 || block == nil || block.Type != "CERTIFICATE REQUEST" || len(rest) != 0 {
		os.Exit(1)
	}
	req, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil || req.CheckSignature() != nil {
		os.Exit(1)
	}
	if _, ok := req.PublicKey.(ed25519.PublicKey); !ok {
		os.Exit(1)
	}
	key, err := x509.MarshalPKIXPublicKey(req.PublicKey)
	if err != nil {
		os.Exit(1)
	}
	digest := sha256.Sum256(req.Raw)
	output := struct {
		Digest    string `json:"digest"`
		PublicKey string `json:"publicKey"`
	}{hex.EncodeToString(digest[:]), string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: key}))}
	if err = json.NewEncoder(os.Stdout).Encode(output); err != nil {
		fmt.Fprintln(os.Stderr, "CSR_OUTPUT")
		os.Exit(1)
	}
}
