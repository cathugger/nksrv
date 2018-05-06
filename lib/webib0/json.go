package webib0

import "encoding/json"

// IBBackReference

var _ json.Marshaler = (*IBBackReference)(nil)
var _ json.Unmarshaler = (*IBBackReference)(nil)

func (r *IBBackReference) MarshalJSON() ([]byte, error) {
	if r.Board == "" && r.Thread == "" {
		return json.Marshal(r.Post)
	} else {
		return json.Marshal(&r.IBReference)
	}
}

func (r *IBBackReference) UnmarshalJSON(b []byte) error {
	e := json.Unmarshal(b, &r.Post)
	if e != nil {
		e = json.Unmarshal(b, &r.IBReference)
		if e != nil {
			return e
		}
	}
	return nil
}

// IBMessage

var _ json.Marshaler = (*IBMessage)(nil)
var _ json.Unmarshaler = (*IBMessage)(nil)

func (m IBMessage) MarshalJSON() ([]byte, error) {
	s := unsafeBytesToStr(m)
	return json.Marshal(s)
}

func (m *IBMessage) UnmarshalJSON(b []byte) error {
	var s string
	e := json.Unmarshal(b, &s)
	if e != nil {
		return e
	}
	*m = unsafeStrToBytes(s)
	return nil
}
