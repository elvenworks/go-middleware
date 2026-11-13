package middleware

import (
	"net/url"
	"strings"
)

const redactedValue = "REDACTED"

// sensitiveQueryKeys are the query parameter names (compared in lower case, after percent-decoding)
// whose values are masked in logs.
var sensitiveQueryKeys = map[string]struct{}{
	"token":         {},
	"access_token":  {},
	"refresh_token": {},
	"id_token":      {},
	"apikey":        {},
	"api_key":       {},
	"key":           {},
	"password":      {},
	"secret":        {},
	"client_secret": {},
	"signature":     {},
	"sig":           {},
}

// RedactQuery returns rawQuery with the value of every sensitive parameter replaced by
// "REDACTED". A parameter can start after '&' or ';'. The value of a sensitive key runs to the
// next '&', so a raw ';' inside it is masked too. Key matching is case-insensitive after
// percent-decoding, and repeated keys are all masked. A part without '=' has no value and is
// left unchanged. Separators, order and every byte outside a masked value are preserved.
func RedactQuery(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(rawQuery))
	for start := 0; start <= len(rawQuery); {
		end := start
		for end < len(rawQuery) && rawQuery[end] != '&' && rawQuery[end] != ';' {
			end++
		}
		part := rawQuery[start:end]
		if key, ok := sensitiveKey(part); ok {
			// The value of a sensitive key ends only at the next '&': a raw ';' is part of it.
			if amp := strings.IndexByte(rawQuery[end:], '&'); amp < 0 {
				end = len(rawQuery)
			} else {
				end += amp
			}
			b.WriteString(key + "=" + redactedValue)
		} else {
			b.WriteString(part)
		}
		if end < len(rawQuery) {
			b.WriteByte(rawQuery[end])
		}
		start = end + 1
	}
	return b.String()
}

// sensitiveKey reports whether part is a "key=value" pair whose key is sensitive, and returns the
// key as written in part.
func sensitiveKey(part string) (rawKey string, ok bool) {
	eq := strings.IndexByte(part, '=')
	if eq < 0 {
		return "", false
	}
	rawKey = part[:eq]
	name, err := url.QueryUnescape(rawKey)
	if err != nil {
		name = rawKey
	}
	_, ok = sensitiveQueryKeys[strings.ToLower(name)]
	return rawKey, ok
}

// RedactRequestURI returns u.RequestURI() with the values of sensitive query parameters
// replaced by "REDACTED", for use in logs. See RedactQuery for the matching rules.
func RedactRequestURI(u *url.URL) string {
	uri := u.RequestURI()
	i := strings.IndexByte(uri, '?')
	if i < 0 {
		return uri
	}
	return uri[:i+1] + RedactQuery(uri[i+1:])
}
