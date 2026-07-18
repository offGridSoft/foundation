// Command keygen mints one key in the exact canonical form Foundation's ingress
// accepts: an Ed25519 offline-authority signing key (-kind ed25519), a
// symmetric random secret (-kind secret, for HMAC / AEAD / CSRF / pepper), or
// a release-obfuscation custody root (-kind garble). Key material is
// written only to a caller-named file (0600); an ed25519 key's public hex prints
// to stdout. It never writes secret material to the terminal and holds no key
// material after exit.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/offGridSoft/foundation/v2026/core"
	foundationkeygen "github.com/offGridSoft/foundation/v2026/keygen"
)

const privateMaterialFileMode os.FileMode = 0o600

var errPrivateMaterialFile = errors.New("keygen private material file")

func main() {
	kindUsage := fmt.Sprintf("key kind: %s | %s | %s", core.KeygenKindTokenEd25519, core.KeygenKindTokenSecret, core.KeygenKindTokenGarble)
	kind := flag.String("kind", "", kindUsage)
	out := flag.String("out", "", "file path to write the private key / secret (created 0600)")
	bytesLen := flag.Int("bytes", core.SecretByteStandard, "secret width in bytes (kind=secret only)")
	flag.Parse()

	if *out == "" {
		fmt.Fprintln(os.Stderr, "keygen: -out is required; key material must go to a file, never the terminal")
		os.Exit(2)
	}
	parsedKind, err := core.ParseKeygenKind(*kind)
	if err != nil {
		fmt.Fprintf(os.Stderr, "keygen: -kind must be %q, %q, or %q\n", core.KeygenKindTokenEd25519, core.KeygenKindTokenSecret, core.KeygenKindTokenGarble)
		os.Exit(2)
	}

	var genErr error
	switch parsedKind {
	case core.KeygenKindEd25519:
		genErr = writeSigningKey(*out)
	case core.KeygenKindSecret:
		genErr = writeSecret(*out, *bytesLen)
	case core.KeygenKindGarbleCustody:
		genErr = writeGarbleCustodySeed(*out)
	}
	if genErr != nil {
		fmt.Fprintf(os.Stderr, "keygen: %v\n", genErr)
		os.Exit(1)
	}
}

func writeGarbleCustodySeed(path string) error {
	seed, err := foundationkeygen.GenerateGarbleCustodySeed()
	if err != nil {
		return err
	}
	material, err := seed.MarshalText()
	if err != nil {
		return err
	}
	defer clear(material)
	if err := writePrivateMaterial(path, string(material)); err != nil {
		return err
	}
	fmt.Println("garble custody seed written to the requested owner-only file — store it, then delete the file")
	return nil
}

func writeSigningKey(path string) error {
	key, err := foundationkeygen.GenerateEd25519SigningKey()
	if err != nil {
		return err
	}
	if err := writePrivateMaterial(path, key.PrivateKeyBase64); err != nil {
		return err
	}
	fmt.Println("ed25519 private key written to the requested owner-only file — store it, then delete the file")
	fmt.Printf("public key (hex): %s\n", key.PublicKeyHex.String())
	return nil
}

func writeSecret(path string, byteLen int) error {
	secret, err := foundationkeygen.GenerateSecretHex(byteLen)
	if err != nil {
		return err
	}
	if err := writePrivateMaterial(path, secret.Hex); err != nil {
		return err
	}
	fmt.Printf("%d-byte secret written to the requested owner-only file — store it, then delete the file\n", secret.ByteLen)
	return nil
}

// writePrivateMaterial creates exactly one new owner-only file. O_EXCL rejects
// existing paths and symlinks, so key generation can never overwrite a caller's
// file or follow a link into an unintended target.
func writePrivateMaterial(path, material string) error {
	var result error
	func() {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, privateMaterialFileMode) // #nosec G304 -- operator-selected output path; exclusive creation is the security boundary.
		if err != nil {
			result = privateMaterialError(err)
			return
		}
		defer func() {
			if err := f.Close(); err != nil {
				result = errors.Join(result, privateMaterialError(err))
			}
			if result != nil {
				result = removePrivateMaterial(path, result)
			}
		}()
		if _, err := io.WriteString(f, material); err != nil {
			result = privateMaterialError(err)
			return
		}
		if err := f.Sync(); err != nil {
			result = privateMaterialError(err)
		}
	}()
	return result
}

func privateMaterialError(err error) error {
	return fmt.Errorf("%w: %w", errPrivateMaterialFile, err)
}

func removePrivateMaterial(path string, cause error) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.Join(cause, privateMaterialError(err))
	}
	return cause
}
