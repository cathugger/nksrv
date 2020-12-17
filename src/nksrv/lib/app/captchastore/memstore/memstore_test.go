package memstore

import (
	"bytes"
	"testing"
)

func TestMemStore(t *testing.T) {
	ms := NewMemStore()
	tk := []byte("kek")
	keks, err := ms.LoadKEKs(func() (id uint64, kek []byte) {
		return 1, tk
	})
	if err != nil {
		t.Errorf("LoadKEKs err: %v", err)
	}
	if len(keks) != 1 || !bytes.Equal(tk, keks[0].KEK) || keks[0].ID != 1 {
		t.Errorf("something happened")
	}

	fresh, err := ms.StoreSolved([]byte("a1"), 5, 1)
	if err != nil {
		t.Errorf("StoreSolved err: %v", err)
	}
	if fresh != true {
		t.Errorf("1")
	}

	fresh, err = ms.StoreSolved([]byte("a1"), 5, 1)
	if err != nil {
		t.Errorf("StoreSolved err: %v", err)
	}
	if fresh != false {
		t.Errorf("2")
	}

	fresh, err = ms.StoreSolved([]byte("a2"), 10, 6)
	if err != nil {
		t.Errorf("StoreSolved err: %v", err)
	}
	if fresh != true {
		t.Errorf("3")
	}

	fresh, err = ms.StoreSolved([]byte("a1"), 15, 6)
	if err != nil {
		t.Errorf("StoreSolved err: %v", err)
	}
	if fresh != true {
		t.Errorf("4")
	}
}
