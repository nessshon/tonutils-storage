package storage

import (
	"bytes"
	"crypto/ed25519"
	"sync/atomic"
	"testing"

	"github.com/xssnick/tonutils-go/adnl"
)

func startedGate(t *testing.T) *adnl.Gateway {
	t.Helper()
	_, key, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate gate key: %v", err)
	}
	gate := adnl.NewGateway(key)
	if err = gate.StartClient(1); err != nil {
		t.Fatalf("failed to start gate client: %v", err)
	}
	return gate
}

func TestPerBagDownloadIdentity(t *testing.T) {
	mainGate := startedGate(t)
	defer mainGate.Close()

	_, key, _ := ed25519.GenerateKey(nil)

	var mints atomic.Int64
	factory := func() (*adnl.Gateway, error) {
		g := startedGate(t)
		mints.Add(1)
		return g, nil
	}

	srv := NewServer(nil, mainGate, factory, key, false, 1)
	defer srv.Stop()

	bagA := bytes.Repeat([]byte{0xA1}, 32)
	bagB := bytes.Repeat([]byte{0xB2}, 32)

	idA := srv.dlIdentityFor(bagA)
	if idA == nil {
		t.Fatal("expected a download identity for bag A")
	}
	if bytes.Equal(idA.gate.GetID(), mainGate.GetID()) {
		t.Fatal("download identity must not reuse the main gate id")
	}

	if srv.dlIdentityFor(bagA) != idA {
		t.Fatal("expected the same download identity to be reused within a bag")
	}

	idB := srv.dlIdentityFor(bagB)
	if idB == idA {
		t.Fatal("expected a distinct download identity for a different bag")
	}
	if bytes.Equal(idA.gate.GetID(), idB.gate.GetID()) {
		t.Fatal("per-bag download identities must have distinct ids")
	}
	if got := mints.Load(); got != 2 {
		t.Fatalf("expected 2 minted gates, got %d", got)
	}

	srv.reapDLIdentity(bagA)
	idA2 := srv.dlIdentityFor(bagA)
	if idA2 == idA {
		t.Fatal("expected a fresh download identity after a reap")
	}
	if got := mints.Load(); got != 3 {
		t.Fatalf("expected 3 minted gates after reap, got %d", got)
	}
}

func TestDownloadIdentityAbsentWithoutFactory(t *testing.T) {
	mainGate := startedGate(t)
	defer mainGate.Close()

	_, key, _ := ed25519.GenerateKey(nil)
	srv := NewServer(nil, mainGate, nil, key, false, 1)
	defer srv.Stop()

	if srv.dlIdentityFor(bytes.Repeat([]byte{0x01}, 32)) != nil {
		t.Fatal("expected no download identity when no factory is configured")
	}
}
