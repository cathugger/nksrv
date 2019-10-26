package psqlib

import (
	"fmt"
	"strconv"
)

type cap_type uint16

const (
	cap_setpriv cap_type = 1 << iota
	cap_delpost
	cap_delboardpost
	cap_delboard
	_
	_
	_
	_
	_
	_
	_
	_

	capx_bits       int = iota
	capx_del            = cap_delpost | cap_delboard
	capx_onlyglobal     = cap_delboard
)

func (c cap_type) String() string {
	var buf [capx_bits]byte
	for i := range buf {
		buf[i] = "01"[(c>>i)&1]
	}
	return string(buf[:])
}

func StrToCap(s string) (c cap_type) {
	if len(s) != capx_bits {
		panic("StrToCap: wrong length")
	}
	for i := 0; i < capx_bits; i++ {
		ch := s[i]
		if ch == '0' {
			// no action need to be taken as zeros are default
		} else if ch == '1' {
			// set bit
			c |= 1 << i
		} else {
			panic("StrToCap: invalid character")
		}
	}
	return
}

type ModCap struct {
	Cap cap_type

	// power for D commands
	DPriv int16
}

func (c ModCap) String() string {
	x := map[string]interface{}{"cap": c.Cap.String()}
	if c.DPriv >= 0 {
		x["dpriv"] = c.DPriv
	}
	return fmt.Sprintf("%#v", x)
}

// MorePriv tells whether o is privileged more by even
// one relevant capability bit or relevant privilege level
func (c ModCap) MorePriv(o ModCap, mask cap_type) bool {
	cc, oc := c.Cap&mask, o.Cap&mask
	if cc|oc != cc {
		// oc filled in some cc gaps
		return true
	}

	// now, they can be the same, or oc can be lesser
	// check relevant privs

	// -1 means not set; lesser wins; therefore cast to uint to make -1 max val
	if (mask&capx_del) != 0 && uint16(o.DPriv) < uint16(c.DPriv) {
		return true
	}

	// well, we failed to prove that o is more at this point
	return false
}

func (c ModCap) Merge(o ModCap) (r ModCap) {

	r.Cap = c.Cap | o.Cap

	// XXX move out to its own func if we have more of these
	if uint16(c.DPriv) <= uint16(o.DPriv) {
		r.DPriv = c.DPriv
	} else {
		r.DPriv = o.DPriv
	}

	return
}

type ModBoardCap map[string]ModCap

func (c ModBoardCap) TakeIn(
	caps map[string]string, dprivs map[string]string) {

	for k, scap := range caps {
		mc := ModCap{Cap: StrToCap(scap)}
		sdpriv := dprivs[k]
		if sdpriv != "" {
			dpriv, err := strconv.ParseUint(sdpriv, 10, 15)
			mc.DPriv = int16(dpriv)
			if err != nil {
				panic("strconv.ParseUint err: " + err.Error())
			}
		} else {
			mc.DPriv = -1
		}
		c[k] = mc
	}
}
