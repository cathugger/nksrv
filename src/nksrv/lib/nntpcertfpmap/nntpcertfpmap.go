package certfpmap

import (
	"bytes"
	"crypto/x509"
	"errors"
	"sort"

	. "nksrv/lib/certfp"
	"nksrv/lib/nntp"
	upn "nksrv/lib/userpassnorm"
)

type fpKeyParam struct {
	sel Selector
	mt  MatchingType
}

type fpKey struct {
	fpKeyParam

	data string
}

type node struct {
	fpKeyParam

	data []byte

	nntp.UserInfo
}

type CertFPMap struct {
	byFingerprint   map[fpKey]*node
	byAnchor        map[string][]*node
	fingerprintVars []fpKeyParam
}

func NewCertFPMap() CertFPMap {
	return CertFPMap{
		byFingerprint: make(map[fpKey]*node),
		byAnchor:      make(map[string][]*node),
	}
}

func (m *CertFPMap) NNTPUserByFingerprint(cert *x509.Certificate) *nntp.UserInfo {
	// we need to go thru every variation we support
	for _, par := range m.fingerprintVars {
		fp := MakeFingerprint(cert, par.sel, par.mt)
		k := fpKey{
			fpKeyParam: fpKeyParam{
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

func (m *CertFPMap) NNTPUserByAnchor(
	cert *x509.Certificate, anchor string) *nntp.UserInfo {

	anchor, err := upn.NormaliseUser(anchor)
	if err != nil {
		return nil
	}

	nodes := m.byAnchor[anchor]
	k := fpKeyParam{sel: -1}
	var fp []byte
	for _, n := range nodes {
		if n.fpKeyParam != k {
			k = n.fpKeyParam
			fp = MakeFingerprint(cert, k.sel, k.mt)
		}
		if bytes.Equal(n.data, fp) {
			return &n.UserInfo
		}
	}
	return nil
}

var _ nntp.CertFPProvider = (*CertFPMap)(nil)

func (m *CertFPMap) Add(
	sel Selector, certfp string, ui nntp.UserInfo) (err error) {

	mt, d, err := ParseCertFP(certfp)
	if err != nil {
		return
	}
	kp := fpKeyParam{
		sel: sel,
		mt:  mt,
	}
	if ui.Name != "" {
		ui.Name, err = upn.NormaliseUser(ui.Name)
		if err != nil {
			return
		}
	}
	n := &node{
		fpKeyParam: kp,
		data:       d,
		UserInfo:   ui,
	}
	k := fpKey{
		fpKeyParam: kp,
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
