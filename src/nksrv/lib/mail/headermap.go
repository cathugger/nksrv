package mail

// if any of these are changed/added/deleted,
// HeaderMapVersion MUST be bumped

const HeaderMapVersion = 1

var headerMap = map[string]string{
	// overrides
	// RFCs digestion, also observation of actual messages
	"Message-Id":        "Message-ID",
	"Content-Id":        "Content-ID",
	"List-Id":           "List-ID",
	"Mime-Version":      "MIME-Version",
	"Nntp-Posting-Date": "NNTP-Posting-Date",
	"Nntp-Posting-Host": "NNTP-Posting-Host",
	// overchan
	"X-Pubkey-Ed25519":            "X-PubKey-Ed25519",
	"X-Signature-Ed25519-Sha512":  "X-Signature-Ed25519-SHA512",
	"X-Signature-Ed25519-Blake2b": "X-Signature-Ed25519-BLAKE2b",
	"X-Frontend-Pubkey":           "X-Frontend-PubKey", // signature below
	"X-Encrypted-Ip":              "X-Encrypted-IP",
	"X-I2p-Desthash":              "X-I2P-DestHash",
}

// TODO do proper analysis what need to be included
