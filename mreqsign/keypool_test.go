package mreqsign

import (
	"testing"
	"time"
)

func TestNewKeyPool_Errors(t *testing.T) {
	if _, err := NewKeyPool(nil, time.Minute); err == nil {
		t.Fatal("expected error for empty key list")
	}
	if _, err := NewKeyPool([][]byte{[]byte("")}, time.Minute); err == nil {
		t.Fatal("expected error for empty key")
	}
	if _, err := NewKeyPool([][]byte{[]byte("key")}, 0); err == nil {
		t.Fatal("expected error for zero window")
	}
}

func TestKeyPool_KeyAt_RotatesByWindow(t *testing.T) {
	keys := [][]byte{[]byte("k0"), []byte("k1"), []byte("k2")}
	pool, err := NewKeyPool(keys, time.Minute, WithEpoch(time.Unix(0, 0)))
	if err != nil {
		t.Fatal(err)
	}

	slot0, key0 := pool.KeyAt(time.Unix(0, 0))
	if slot0 != 0 || string(key0) != "k0" {
		t.Fatalf("got slot=%d key=%s, want slot=0 key=k0", slot0, key0)
	}

	slot1, key1 := pool.KeyAt(time.Unix(60, 0))
	if slot1 != 1 || string(key1) != "k1" {
		t.Fatalf("got slot=%d key=%s, want slot=1 key=k1", slot1, key1)
	}

	// wraps around the pool after 3 windows
	slot3, key3 := pool.KeyAt(time.Unix(180, 0))
	if slot3 != 3 || string(key3) != "k0" {
		t.Fatalf("got slot=%d key=%s, want slot=3 key=k0", slot3, key3)
	}
}

func TestGenerateKeys(t *testing.T) {
	seed := []byte("shared-secret-seed")
	keys, err := GenerateKeys(seed, 4, 32)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 4 {
		t.Fatalf("got %d keys, want 4", len(keys))
	}
	for _, k := range keys {
		if len(k) != 32 {
			t.Fatalf("got key length %d, want 32", len(k))
		}
	}
}

func TestGenerateKeys_Deterministic(t *testing.T) {
	seed := []byte("shared-secret-seed")

	keysA, err := GenerateKeys(seed, 5, 24)
	if err != nil {
		t.Fatal(err)
	}
	keysB, err := GenerateKeys(seed, 5, 24)
	if err != nil {
		t.Fatal(err)
	}

	for i := range keysA {
		if string(keysA[i]) != string(keysB[i]) {
			t.Fatalf("key %d differs between independent calls with the same seed", i)
		}
	}
}

func TestGenerateKeys_DifferentSeedDiffers(t *testing.T) {
	keysA, _ := GenerateKeys([]byte("seed-a"), 3, 32)
	keysB, _ := GenerateKeys([]byte("seed-b"), 3, 32)

	for i := range keysA {
		if string(keysA[i]) == string(keysB[i]) {
			t.Fatalf("key %d unexpectedly matches for different seeds", i)
		}
	}
}

func TestGenerateKeys_Errors(t *testing.T) {
	if _, err := GenerateKeys(nil, 1, 32); err == nil {
		t.Fatal("expected error for empty seed")
	}
	if _, err := GenerateKeys([]byte("seed"), 0, 32); err == nil {
		t.Fatal("expected error for n <= 0")
	}
	if _, err := GenerateKeys([]byte("seed"), 1, 0); err == nil {
		t.Fatal("expected error for keyLen <= 0")
	}
}
