package core

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
)

type signedTestBody struct {
	Value string `json:"value"`
}

func (b signedTestBody) Validate() error {
	if b.Value == "" {
		return ErrFoundationContract
	}
	return nil
}

func (b signedTestBody) Canonical(dst []byte) ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return AppendCanonicalJSON(dst, b)
}

func TestSignedVerifyHostileTable(t *testing.T) {
	t.Parallel()
	keyID, publicKey, privateKey := signingTestKey(t, "server-key-1")
	otherID, otherPublicKey, _ := signingTestKey(t, "server-key-2")
	body := signedTestBody{Value: "ok"}
	message, err := body.Canonical(nil)
	if err != nil {
		t.Fatal(err)
	}
	signature := ed25519.Sign(privateKey, message)
	for _, tc := range []struct {
		name    string
		signed  Signed[signedTestBody]
		keyring SigningKeyring
	}{
		{
			name: "tampered body rejected",
			signed: Signed[signedTestBody]{
				KeyID:     keyID,
				Signature: signature,
				Body:      signedTestBody{Value: "changed"},
			},
			keyring: SigningKeyring{Keys: []SigningPublicKey{{ID: keyID, PublicKey: publicKey}}},
		},
		{
			name: "flipped signature rejected",
			signed: Signed[signedTestBody]{
				KeyID:     keyID,
				Signature: flipSignature(signature),
				Body:      body,
			},
			keyring: SigningKeyring{Keys: []SigningPublicKey{{ID: keyID, PublicKey: publicKey}}},
		},
		{
			name: "unknown key id rejected",
			signed: Signed[signedTestBody]{
				KeyID:     otherID,
				Signature: signature,
				Body:      body,
			},
			keyring: SigningKeyring{Keys: []SigningPublicKey{{ID: keyID, PublicKey: publicKey}}},
		},
		{
			name: "wrong public key rejected",
			signed: Signed[signedTestBody]{
				KeyID:     keyID,
				Signature: signature,
				Body:      body,
			},
			keyring: SigningKeyring{Keys: []SigningPublicKey{{ID: keyID, PublicKey: otherPublicKey}}},
		},
		{
			name: "empty signature rejected",
			signed: Signed[signedTestBody]{
				KeyID: keyID,
				Body:  body,
			},
			keyring: SigningKeyring{Keys: []SigningPublicKey{{ID: keyID, PublicKey: publicKey}}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.signed.Verify(tc.keyring); !errors.Is(err, ErrFoundationContract) {
				t.Fatalf("Verify error = %v, want ErrFoundationContract", err)
			}
		})
	}
}

func TestSignedVerifyAcceptsValidSignature(t *testing.T) {
	t.Parallel()
	keyID, publicKey, privateKey := signingTestKey(t, "server-key-1")
	body := signedTestBody{Value: "ok"}
	message, err := body.Canonical(nil)
	if err != nil {
		t.Fatal(err)
	}
	signed := Signed[signedTestBody]{
		KeyID:     keyID,
		Signature: ed25519.Sign(privateKey, message),
		Body:      body,
	}
	keyring := SigningKeyring{Keys: []SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
	if err := signed.Verify(keyring); err != nil {
		t.Fatal(err)
	}
}

func signingTestKey(t *testing.T, id string) (SigningKeyID, Ed25519PublicKeyHex, ed25519.PrivateKey) {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keyID, err := ParseSigningKeyID(id)
	if err != nil {
		t.Fatal(err)
	}
	publicHex, err := NewEd25519PublicKeyHex(public)
	if err != nil {
		t.Fatal(err)
	}
	return keyID, publicHex, private
}

func flipSignature(signature []byte) []byte {
	out := append([]byte(nil), signature...)
	out[0] ^= 1
	return out
}
