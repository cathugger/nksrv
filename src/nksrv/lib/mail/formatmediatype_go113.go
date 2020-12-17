// +build go1.13

package mail

import "mime"

func FormatMediaTypeX(t string, param map[string]string) string {
	return mime.FormatMediaType(t, param)
}
