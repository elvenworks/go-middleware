package middleware

import (
	"net/url"
	"testing"
)

var redactQueryCases = []struct {
	name string
	in   string
	want string
}{
	{"empty", "", ""},
	{"no sensitive keys", "page=2&size=10", "page=2&size=10"},
	{"token first", "token=abc&orgUid=o1", "token=REDACTED&orgUid=o1"},
	{"upper case key", "orgUid=o1&TOKEN=abc", "orgUid=o1&TOKEN=REDACTED"},
	{"percent-encoded key", "%74oken=abc", "%74oken=REDACTED"},
	{"hex letter escape in key", "t%6Fken=abc", "t%6Fken=REDACTED"},
	{"escape and upper case in key", "%54OKEN=abc", "%54OKEN=REDACTED"},
	{"plus in key is a space", "tok+en=abc&token+=abc", "tok+en=abc&token+=abc"},
	{"value starting with equals", "token==abc", "token=REDACTED"},
	{"hash kept in raw query", "x=1#&token=abc", "x=1#&token=REDACTED"},
	{"repeated key", "token=a&token=b", "token=REDACTED&token=REDACTED"},
	{"semicolon separator", "a=1;token=x", "a=1;token=REDACTED"},
	{"semicolon inside value", "token=a;b&x=1", "token=REDACTED&x=1"},
	{"semicolon inside last value", "x=1&token=a;b", "x=1&token=REDACTED"},
	{"semicolon after value is masked too", "token=x;a=1&b=2", "token=REDACTED&b=2"},
	{"key without value", "token", "token"},
	{"empty value", "token=", "token=REDACTED"},
	{"encoded ampersand in value", "token=a%26b&x=1", "token=REDACTED&x=1"},
	{"invalid escape in other key", "%zz=1&token=x", "%zz=1&token=REDACTED"},
	{"key must match exactly", "keyword=x&tokens=y", "keyword=x&tokens=y"},
	{
		"all sensitive keys",
		"api_key=1&apikey=2&key=3&password=4&secret=5&client_secret=6&access_token=7&refresh_token=8&id_token=9&signature=10&sig=11",
		"api_key=REDACTED&apikey=REDACTED&key=REDACTED&password=REDACTED&secret=REDACTED&client_secret=REDACTED&access_token=REDACTED&refresh_token=REDACTED&id_token=REDACTED&signature=REDACTED&sig=REDACTED",
	},
}

func TestRedactQuery(t *testing.T) {
	for _, tc := range redactQueryCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RedactQuery(tc.in); got != tc.want {
				t.Fatalf("RedactQuery(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestRedactQueryIdempotent(t *testing.T) {
	for _, tc := range redactQueryCases {
		t.Run(tc.name, func(t *testing.T) {
			once := RedactQuery(tc.in)
			if twice := RedactQuery(once); twice != once {
				t.Fatalf("RedactQuery not idempotent: %q then %q", once, twice)
			}
		})
	}
}

func TestRedactRequestURI(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/ok?token=abc&x=1", "/ok?token=REDACTED&x=1"},
		{"/ok", "/ok"},
		{"/a%2Fb?token=x", "/a%2Fb?token=REDACTED"},
		{"http://h/ok?token=x", "/ok?token=REDACTED"},
	}
	for _, tc := range cases {
		// net/http parses the request target of a server request with url.ParseRequestURI.
		u, err := url.ParseRequestURI(tc.in)
		if err != nil {
			t.Fatal(err)
		}
		if got := RedactRequestURI(u); got != tc.want {
			t.Fatalf("RedactRequestURI(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
