package captcha

import (
	"bytes"
	"testing"
)

func TestPackBCD(t *testing.T) {
	type xs struct{ unp, pak []byte }
	x := [...]xs{
		xs{[]byte{}, []byte{}},
		xs{[]byte{0x00}, []byte{0x00}},
		xs{[]byte{0x09}, []byte{0x09}},
		xs{[]byte{0x01, 0x02}, []byte{0x12}},
		xs{[]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x23}},
		xs{[]byte{0x01, 0x02, 0x03, 0x04}, []byte{0x12, 0x34}},
	}
	for i := range x {
		res, err := packBCD(nil, x[i].unp)
		if err != nil {
			t.Errorf("%d: err: %v", i, err)
		}
		if !bytes.Equal(res, x[i].pak) {
			t.Errorf("%d: expected %v got %v", i, x[i].pak, res)
		}
	}
}

func TestUnpackBCD(t *testing.T) {
	type xs struct {
		unp, pak []byte
		xlen     int
	}
	x := [...]xs{
		xs{[]byte{}, []byte{}, 0},
		xs{[]byte{0x00}, []byte{0x00}, 1},
		xs{[]byte{0x09}, []byte{0x09}, 1},
		xs{[]byte{0x01, 0x02}, []byte{0x12}, 2},
		xs{[]byte{0x01, 0x02, 0x03}, []byte{0x01, 0x23}, 3},
		xs{[]byte{0x01, 0x02, 0x03}, []byte{0x71, 0x23}, 3},
		xs{[]byte{0x01, 0x02, 0x03, 0x04}, []byte{0x12, 0x34}, 4},
		xs{[]byte{0x02, 0x03, 0x04}, []byte{0x12, 0x34}, 3},
		xs{[]byte{0x03, 0x04}, []byte{0x34, 0x12}, 2},
		xs{[]byte{0x04}, []byte{0x34, 0x12}, 1},
		xs{[]byte{}, []byte{0x34, 0x12}, 0},
	}
	for i := range x {
		res, err := unpackBCD(nil, x[i].pak, x[i].xlen)
		if err != nil {
			t.Errorf("%d: err: %v", i, err)
		}
		if !bytes.Equal(res, x[i].unp) {
			t.Errorf("%d: expected %v got %v", i, x[i].unp, res)
		}
	}
}
