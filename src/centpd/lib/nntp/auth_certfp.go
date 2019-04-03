package nntp

import "crypto/x509"

type CertFPProvider interface {
	// may be a bit costy with a lot of different FP param variations
	NNTPUserByFingerprint(cert *x509.Certificate) *UserInfo

	// anchor can be SASL EXTERNAL identity, or maybe something from subject?
	// in any case it must be non empty
	NNTPUserByAnchor(cert *x509.Certificate, anchor string) *UserInfo
}
