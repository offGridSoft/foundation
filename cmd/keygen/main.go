// Command keygen mints one key in the exact canonical form Foundation's ingress
// accepts: an Ed25519 offline-authority signing key (-kind ed25519) or a
// symmetric random secret (-kind secret, for HMAC / AEAD / CSRF / pepper). Key
// material is written only to a caller-named file (0600); an ed25519 key's
// public hex prints to stdout. It never writes secret material to the terminal
// and holds no key material after exit.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/offGridSoft/foundation/v2026/core"
)

func main() {
	kind := flag.String("kind", "", "key kind: ed25519 | secret")
	out := flag.String("out", "", "file path to write the private key / secret (created 0600)")
	bytesLen := flag.Int("bytes", 32, "secret width in bytes (kind=secret only)")
	flag.Parse()

	if *out == "" {
		fmt.Fprintln(os.Stderr, "keygen: -out is required; key material must go to a file, never the terminal")
		os.Exit(2)
	}
	parsedKind, err := core.ParseKeygenKind(*kind)
	if err != nil {
		fmt.Fprintf(os.Stderr, "keygen: -kind must be %q or %q\n", core.KeygenKindTokenEd25519, core.KeygenKindTokenSecret)
		os.Exit(2)
	}

	var genErr error
	switch parsedKind {
	case core.KeygenKindEd25519:
		genErr = writeSigningKey(*out)
	case core.KeygenKindSecret:
		genErr = writeSecret(*out, *bytesLen)
	}
	if genErr != nil {
		fmt.Fprintf(os.Stderr, "keygen: %v\n", genErr)
		os.Exit(1)
	}
}

func writeSigningKey(path string) error {
	key, err := core.GenerateEd25519SigningKey()
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(key.PrivateKeyBase64+"\n"), 0o600); err != nil {
		return err
	}
	fmt.Printf("ed25519 private key (base64) written to %s (0600) — store it, then delete the file\n", path)
	fmt.Printf("public key (hex): %s\n", key.PublicKeyHex.String())
	return nil
}

func writeSecret(path string, byteLen int) error {
	secret, err := core.GenerateSecretHex(byteLen)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(secret.Hex+"\n"), 0o600); err != nil {
		return err
	}
	fmt.Printf("%d-byte secret (hex) written to %s (0600) — store it, then delete the file\n", secret.ByteLen, path)
	return nil
}
