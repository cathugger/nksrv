package captchastore

// to store solved captchas to prevent reuse

type KEKInfo struct {
	ID       uint64
	KEK      []byte
	Disabled bool
}

type CaptchaStore interface {
	StoreSolved(obj []byte, expires, nowtime int64) (fresh bool, err error)
	LoadKEKs(ifempty func() (id uint64, kek []byte)) (keks []KEKInfo, err error)
}
