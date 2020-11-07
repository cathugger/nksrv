package fstore

import (
	crand "crypto/rand"
	"encoding/binary"
	"os"
	"strconv"
	"sync"
	"time"

	"nksrv/lib/utils/pcg"
)

var (
	rng     pcg.PCG64s
	rngInit bool
	rngMu   sync.Mutex
)

func reseedLocked() {
	var b [16]byte
	if _, e := crand.Read(b[:]); e != nil {
		panic(e.Error())
	}
	x, y := binary.BigEndian.Uint64(b[:8]), binary.BigEndian.Uint64(b[8:])
	x += uint64(os.Getpid())
	y += uint64(time.Now().UnixNano())
	rng.Seed(x, y)
}

func reseed() {
	rngMu.Lock()
	reseedLocked()
	rngMu.Unlock()
}

func nextSuffix() string {
	rngMu.Lock()
	if !rngInit {
		reseedLocked()
		rngInit = true
	}
	x := rng.Bounded(1e18)
	rngMu.Unlock()
	return strconv.FormatUint(1e18+x, 10)[1:]
}
