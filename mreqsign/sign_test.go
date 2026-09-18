package mreqsign

import (
	"testing"
	"time"
)

func TestSignAndVerify_OK(t *testing.T) {
	keys, err := GenerateKeys([]byte("shared-seed"), 3, 32)
	if err != nil {
		t.Fatal(err)
	}
	pool, err := NewKeyPool(keys, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	payload := []byte("GET&/api/order&123")
	now := time.Now()
	ts, sig := pool.Sign(payload, now)

	if err := pool.Verify(payload, ts, sig, 5*time.Second); err != nil {
		t.Fatalf("expected verify to succeed, got %v", err)
	}
}

func TestVerify_TamperedPayload(t *testing.T) {
	keys, _ := GenerateKeys([]byte("shared-seed"), 2, 32)
	pool, _ := NewKeyPool(keys, time.Minute)

	now := time.Now()
	ts, sig := pool.Sign([]byte("original"), now)

	if err := pool.Verify([]byte("tampered"), ts, sig, 5*time.Second); err != ErrInvalidSignature {
		t.Fatalf("got %v, want ErrInvalidSignature", err)
	}
}

func TestVerify_Expired(t *testing.T) {
	keys, _ := GenerateKeys([]byte("shared-seed"), 2, 32)
	pool, _ := NewKeyPool(keys, time.Minute)

	payload := []byte("data")
	old := time.Now().Add(-time.Hour)
	ts, sig := pool.Sign(payload, old)

	if err := pool.Verify(payload, ts, sig, 5*time.Second); err != ErrExpired {
		t.Fatalf("got %v, want ErrExpired", err)
	}
}

func TestVerify_DifferentPoolFails(t *testing.T) {
	keysA, _ := GenerateKeys([]byte("seed-a"), 2, 32)
	keysB, _ := GenerateKeys([]byte("seed-b"), 2, 32)
	poolA, _ := NewKeyPool(keysA, time.Minute)
	poolB, _ := NewKeyPool(keysB, time.Minute)

	payload := []byte("data")
	now := time.Now()
	ts, sig := poolA.Sign(payload, now)

	if err := poolB.Verify(payload, ts, sig, 5*time.Second); err != ErrInvalidSignature {
		t.Fatalf("got %v, want ErrInvalidSignature", err)
	}
}

func TestSignVerify_AcrossWindowBoundary(t *testing.T) {
	// two independent pools built from the same keys and epoch must derive the
	// same key purely from the timestamp carried in the request, regardless of
	// which rotation window that timestamp falls into.
	keys, _ := GenerateKeys([]byte("shared-seed"), 3, 32)
	epoch := time.Now().Add(-time.Hour).Truncate(time.Minute)
	signerPool, _ := NewKeyPool(keys, time.Minute, WithEpoch(epoch))
	verifierPool, _ := NewKeyPool(keys, time.Minute, WithEpoch(epoch))

	payload := []byte("data")
	justBeforeRotation := epoch.Add(59 * time.Second)
	ts, sig := signerPool.Sign(payload, justBeforeRotation)

	if err := verifierPool.Verify(payload, ts, sig, time.Hour); err != nil {
		t.Fatalf("expected verify to succeed, got %v", err)
	}
}
