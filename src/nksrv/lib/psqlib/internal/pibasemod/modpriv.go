package pibasemod

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

type CapType uint16

const (
	cap_Reserved0 = 1 << iota
	Cap_DelPost
	Cap_DelBoardPost
	Cap_DelBoard
	_
	_
	_
	_
	_
	_
	_
	_

	capX_Bits       int = iota
	CapX_OnlyGlobal     = Cap_DelBoard
)

func (c CapType) String() string {
	var buf [capX_Bits]byte
	for i := range buf {
		buf[i] = "01"[(c>>i)&1]
	}
	return string(buf[:])
}

func StrToCap(s string) (c CapType) {
	if len(s) != capX_Bits {
		panic("StrToCap: wrong length")
	}
	for i := 0; i < capX_Bits; i++ {
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

const (
	CapLvl_DelPost = iota

	CapLvlX_Num
)

var caplvlx_mask = [CapLvlX_Num]CapType{Cap_DelBoard}

type TCapLvl = int16

const CapLvl_MaxVal = 0x7Fff

type ModCap struct {
	Cap CapType

	CapLevel [CapLvlX_Num]TCapLvl // -1 = unset
}

var NoneModCap = ModCap{
	Cap:      0,
	CapLevel: [CapLvlX_Num]TCapLvl{-1},
}

func (c ModCap) String() string {
	x := map[string]interface{}{
		"cap":       c.Cap.String(),
		"cap_level": c.CapLevel,
	}
	return fmt.Sprintf("%#v", x)
}

// MorePriv tells whether o is privileged more by even
// one relevant capability bit or relevant privilege level
func (c ModCap) MorePriv(o ModCap, mask CapType) bool {

	cc, oc := c.Cap&mask, o.Cap&mask
	if cc|oc != cc {
		// oc filled in some cc gaps
		return true
	}

	// now, they can be the same, or oc can be lesser
	// check relevant privs

	for i := range c.CapLevel {
		// -1 becomes max val when converted to uint
		if (mask&caplvlx_mask[i]) != 0 &&
			uint16(o.CapLevel[i]) < uint16(c.CapLevel[i]) {

			return true
		}
	}

	// well, we failed to prove that o is more at this point
	return false
}

func (c ModCap) Merge(o ModCap) (r ModCap) {

	r.Cap = c.Cap | o.Cap

	for i := range r.CapLevel {
		// -1 becomes max val when converted to uint
		if uint16(c.CapLevel[i]) <= uint16(o.CapLevel[i]) {
			r.CapLevel[i] = c.CapLevel[i]
		} else {
			r.CapLevel[i] = o.CapLevel[i]
		}
	}

	return
}

type ModBoardCap map[string]ModCap

type ModCombinedCaps struct {
	ModCap             ModCap
	ModBoardCap        ModBoardCap
	ModInheritCap      ModCap
	ModInheritBoardCap ModBoardCap
}

func ProcessCapLevel(mc ModCap, arr []sql.NullInt32) ModCap {
	for i := range mc.CapLevel {
		if i < len(arr) && arr[i].Valid {
			if uint32(arr[i].Int32) > CapLvl_MaxVal {
				panic("too large val")
			}
			mc.CapLevel[i] = TCapLvl(arr[i].Int32)
		} else {
			mc.CapLevel[i] = -1
		}
	}
	return mc
}

// DB-specific func
func (c ModBoardCap) TakeIn(
	caps map[string]string, caplvls map[string]string) {

	for k, scap := range caps {

		mc := ModCap{Cap: StrToCap(scap)}

		scaplvl := caplvls[k]

		var arr []sql.NullInt32

		if scaplvl != "" {
			err := pq.Array(&arr).Scan(scaplvl)
			if err != nil {
				panic("pq.Array scan err: " + err.Error())
			}
			// XXX is this check needed?
			if len(arr) != len(mc.CapLevel) {
				panic("lengths don't match")
			}
		}

		mc = ProcessCapLevel(mc, arr)

		c[k] = mc
	}
}
