package transformer

import (
	"encoding/base64"
	"regexp"
	"strings"
)

var genericMap = map[string]interface{}{
	// String functions:
	"upper":      strings.ToUpper,
	"lower":      strings.ToLower,
	"trim":       strings.TrimSpace,
	"trimSuffix": func(a, b string) string { return strings.TrimSuffix(b, a) },
	"trimPrefix": func(a, b string) string { return strings.TrimPrefix(b, a) },

	"replace":         func(o, n, s string) string { return strings.ReplaceAll(s, o, n) },
	"regexFind":       regexFind,
	"regexFindString": regexFindString,
	"regexReplaceAll": regexReplaceAll,

	"contains":  func(substr string, str string) bool { return strings.Contains(str, substr) },
	"hasPrefix": func(substr string, str string) bool { return strings.HasPrefix(str, substr) },
	"hasSuffix": func(substr string, str string) bool { return strings.HasSuffix(str, substr) },

	// Encoding functions:
	"b64enc": base64encode,
	"b64dec": base64decode,
}

// GenericFuncMap returns a copy of the basic function map as a map[string]interface{}.
func GenericFuncMap() map[string]interface{} {
	gfm := make(map[string]interface{}, len(genericMap))
	for k, v := range genericMap {
		gfm[k] = v
	}

	return gfm
}

func regexFindString(regex string, s string, n int) (string, error) {
	r, err := regexp.Compile(regex)
	if err != nil {
		return "", err
	}

	matches := r.FindStringSubmatch(s)

	if len(matches) < n+1 {
		return "", nil
	}

	return matches[n], nil
}

func regexReplaceAll(regex string, s string, repl string) (string, error) {
	r, err := regexp.Compile(regex)
	if err != nil {
		return "", err
	}

	return r.ReplaceAllString(s, repl), nil
}

func regexFind(regex string, s string) (string, error) {
	r, err := regexp.Compile(regex)
	if err != nil {
		return "", err
	}

	return r.FindString(s), nil
}

func base64encode(v string) string {
	return base64.StdEncoding.EncodeToString([]byte(v))
}

func base64decode(v string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
