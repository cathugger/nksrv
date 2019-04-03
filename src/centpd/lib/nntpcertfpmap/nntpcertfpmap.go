package certfpmap

import (
	"bytes"
	"crypto/x509"
	"errors"
	"sort"

	. "centpd/lib/certfp"
	"centpd/lib/nntp"
	upn "centpd/lib/userpassnorm"
)

type FPKeyParam struct {
	sel Selector
	mt  MatchingType
}

type FPKey struct {
	FPKeyParam

	data string
}

type Node struct {
	FPKeyParam

	data []byte

	nntp.UserInfo
}

type CertFPMap struct {
	byFingerprint   map[FPKey]*Node
	byAnchor        map[string][]*Node
	fingerprintVars []FPKeyParam
}

func (m *CertFPMap) LookupByFP(cert *x509.Certificate) *nntp.UserInfo {
	// we need to go thru every variation we support
	for _, par := range m.fingerprintVars {
		fp := MakeFingerprint(cert, par.sel, par.mt)
		k := FPKey{
			FPKeyParam: FPKeyParam{
				sel: par.sel,
				mt:  par.mt,
			},
			data: string(fp),
		}
		n := m.byFingerprint[k]
		if n != nil {
			return &n.UserInfo
		}
	}
	return nil
}

func (m *CertFPMap) LookupByAnchor(
	cert *x509.Certificate, anchor string) *nntp.UserInfo {

	anchor, err := upn.NormaliseUser(anchor)
	if err != nil {
		return nil
	}

	nodes := m.byAnchor[anchor]
	k := FPKeyParam{sel: -1}
	var fp []byte
	for _, n := range nodes {
		if n.FPKeyParam != k {
			k = n.FPKeyParam
			fp = MakeFingerprint(cert, k.sel, k.mt)
		}
		if bytes.Equal(n.data, fp) {
			return &n.UserInfo
		}
	}
	return nil
}

func (m *CertFPMap) Add(
	sel Selector, certfp string, ui nntp.UserInfo) (err error) {

	mt, d, err := ParseCertFP(certfp)
	if err != nil {
		return
	}
	kp := FPKeyParam{
		sel: sel,
		mt:  mt,
	}
	if ui.Name != "" {
		ui.Name, err = upn.NormaliseUser(ui.Name)
		if err != nil {
			return
		}
	}
	n := &Node{
		FPKeyParam: kp,
		data:       d,
		UserInfo:   ui,
	}
	k := FPKey{
		FPKeyParam: kp,
		data:       string(d),
	}

	_, exists := m.byFingerprint[k]
	if exists {
		return errors.New("duplicate key")
	}
	if ui.Name != "" {
		ns := m.byAnchor[ui.Name]
		ns = append(ns, n)
		sort.Slice(ns, func(i, j int) bool {
			if ns[i].sel < ns[j].sel {
				return true
			}
			if ns[i].sel > ns[j].sel {
				return false
			}
			return ns[i].mt < ns[j].mt
		})
		m.byAnchor[ui.Name] = ns
	}

	m.byFingerprint[k] = n

	return nil
}
