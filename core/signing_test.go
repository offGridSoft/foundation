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
	message, err := AppendSignedMessage(nil, keyID, body)
	if err != nil {
		t.Fatal(err)
	}
	signature := signTestMessage(t, privateKey, message)
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
				Signature: flipSignature(t, signature),
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
	message, err := AppendSignedMessage(nil, keyID, body)
	if err != nil {
		t.Fatal(err)
	}
	signed := Signed[signedTestBody]{
		KeyID:     keyID,
		Signature: signTestMessage(t, privateKey, message),
		Body:      body,
	}
	keyring := SigningKeyring{Keys: []SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
	if err := signed.Verify(keyring); err != nil {
		t.Fatal(err)
	}
}

func TestAppendSignedMessageWireLayout(t *testing.T) {
	t.Parallel()
	keyID, _, _ := signingTestKey(t, "server-key-1")
	got, err := AppendSignedMessage(nil, keyID, signedTestBody{Value: "ok"})
	if err != nil {
		t.Fatal(err)
	}
	want := "foundation-signed-v1\x00server-key-1\x00{\"value\":\"ok\"}"
	if string(got) != want {
		t.Fatalf("AppendSignedMessage() = %q, want %q", got, want)
	}
}

func TestSignedVerifyRejectsSignatureBoundToDifferentKeyID(t *testing.T) {
	t.Parallel()
	keyID, publicKey, privateKey := signingTestKey(t, "server-key-1")
	otherID, _, _ := signingTestKey(t, "server-key-2")
	body := signedTestBody{Value: "ok"}
	message, err := AppendSignedMessage(nil, otherID, body)
	if err != nil {
		t.Fatal(err)
	}
	signed := Signed[signedTestBody]{
		KeyID:     keyID,
		Signature: signTestMessage(t, privateKey, message),
		Body:      body,
	}
	keyring := SigningKeyring{Keys: []SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
	if err := signed.Verify(keyring); !errors.Is(err, ErrFoundationContract) {
		t.Fatalf("Verify wrong-key-id-bound signature error = %v, want ErrFoundationContract", err)
	}
}

func TestSigningKeyringRejectsDuplicateKeyID(t *testing.T) {
	t.Parallel()
	keyID, publicKey, _ := signingTestKey(t, "server-key-1")
	ring := SigningKeyring{Keys: []SigningPublicKey{
		{ID: keyID, PublicKey: publicKey},
		{ID: keyID, PublicKey: publicKey},
	}}
	if err := ring.Validate(); !errors.Is(err, ErrFoundationContract) {
		t.Fatalf("SigningKeyring duplicate Validate error = %v, want ErrFoundationContract", err)
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

func signTestMessage(t *testing.T, privateKey ed25519.PrivateKey, message []byte) Ed25519SignatureHex {
	t.Helper()
	signature, err := NewEd25519SignatureHex(ed25519.Sign(privateKey, message))
	if err != nil {
		t.Fatal(err)
	}
	return signature
}

func flipSignature(t *testing.T, signature Ed25519SignatureHex) Ed25519SignatureHex {
	t.Helper()
	raw, err := signature.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	out := append([]byte(nil), raw...)
	out[0] ^= 1
	flipped, err := NewEd25519SignatureHex(out)
	if err != nil {
		t.Fatal(err)
	}
	return flipped
}
