package memstore

import (
	"sync"

	"nksrv/lib/app/captchastore"
)

var _ captchastore.CaptchaStore = (*MemStore)(nil)

type expobj struct {
	next, prev *expobj
	obj        []byte
	exp        int64
}

type MemStore struct {
	lock       sync.Mutex
	keks       []captchastore.KEKInfo
	solved     map[string]*expobj
	head, tail *expobj
}

func NewMemStore() captchastore.CaptchaStore {
	ms := &MemStore{
		solved: make(map[string]*expobj),
	}
	return ms
}

func (ms *MemStore) StoreSolved(
	obj []byte, expires, nowtime int64) (fresh bool, err error) {

	ms.lock.Lock()
	defer ms.lock.Unlock()

	for xp := ms.tail; xp != nil && xp.exp < nowtime; xp = xp.prev {
		ms.tail = xp.prev
		if xp.prev != nil {
			xp.prev.next = nil
		}
		delete(ms.solved, string(xp.obj))
	}

	_, exists := ms.solved[string(obj)]
	if exists {
		return false, nil
	}

	cp := make([]byte, len(obj))
	copy(cp, obj)
	eo := &expobj{
		obj: cp,
		exp: expires,
	}

	// insert into list
	var lastp *expobj
	nextpp := &ms.head
	for {
		// something like this I don't remember
		// >. .A. .B. .C. .<
		if *nextpp == nil || (*nextpp).exp <= expires {
			// in C this would look prettier..
			var prevpp **expobj
			if *nextpp != nil {
				prevpp = &(*nextpp).prev
			} else {
				prevpp = &ms.tail
			}

			// proper links of object itself
			eo.next = *nextpp // we insert this before next
			eo.prev = lastp   // object which holds *nextpp if any

			// now link object into surroundings
			*prevpp = eo
			*nextpp = eo

			break
		}
		lastp = *nextpp
	}

	ms.solved[string(obj)] = eo

	return true, nil
}

func (ms *MemStore) LoadKEKs(
	ifempty func() (id uint64, kek []byte)) (
	keks []captchastore.KEKInfo, err error) {

	ms.lock.Lock()
	defer ms.lock.Unlock()

	if len(ms.keks) == 0 && ifempty != nil {
		id, kek := ifempty()
		ms.keks = append(ms.keks, captchastore.KEKInfo{ID: id, KEK: kek})
	}
	return ms.keks, nil
}
