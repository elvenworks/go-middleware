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
// "REDACTED". Parts are split on '&' and ';'; separators, order and every non-sensitive byte are
// preserved. Key matching is case-insensitive and percent-decoded, and applies to repeated keys.
// A part without '=' has no value and is left unchanged.
func RedactQuery(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(rawQuery))
	start := 0
	for i := 0; i <= len(rawQuery); i++ {
		if i < len(rawQuery) && rawQuery[i] != '&' && rawQuery[i] != ';' {
			continue
		}
		b.WriteString(redactPart(rawQuery[start:i]))
		if i < len(rawQuery) {
			b.WriteByte(rawQuery[i])
		}
		start = i + 1
	}
	return b.String()
}

func redactPart(part string) string {
	eq := strings.IndexByte(part, '=')
	if eq < 0 {
		return part
	}
	key := part[:eq]
	name, err := url.QueryUnescape(key)
	if err != nil {
		name = key
	}
	if _, ok := sensitiveQueryKeys[strings.ToLower(name)]; ok {
		return key + "=" + redactedValue
	}
	return part
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
