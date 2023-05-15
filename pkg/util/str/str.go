package str

import (
	"strings"
	"unicode"
)

func ToCamel(str string) string {
	if len(str) < 1 {
		return ""
	}
	strArry := []rune(str)
	if strArry[0] >= 97 && strArry[0] <= 122 {
		strArry[0] -= 32
	}
	return string(strArry)
}

func ToPlural(str string) string {
	lastChar := str[len(str)-1:]
	if lastChar == "s" {
		return str + "es"
	} else {
		return str + "s"
	}
}

func ToShort(str string) string {
	if IsSnakeCase(ToSnakeCase(str)) {
		str = ToSnakeCase(str)
	}

	if IsSnakeCase(str) {
		parts := strings.Split(str, "_")
		return parts[0][0:1] + parts[1][0:1]
	}

	return strings.ToLower(str[0:1])
}

func ToLowerCamelCase(in string) string {
	runes := []rune(in)

	var out []rune
	flag := false
	for i, curr := range runes {
		if (i == 0 && unicode.IsUpper(curr)) || (flag && unicode.IsUpper(curr)) {
			out = append(out, unicode.ToLower(curr))
			flag = true
		} else {
			out = append(out, curr)
			flag = false
		}
	}

	return string(out)
}

func SnakeToLowerCamel(in string) string {
	camel := SnakeToCamel(in)

	return strings.ToLower(camel[0:1]) + camel[1:]
}

func ToPascal(in string) string {
	if u := strings.ToUpper(in); commonInitialisms[u] {
		return u
	}

	if IsSnakeCase(in) {
		return SnakeToCamel(in)
	}

	return strings.Title(in)
}

func SnakeToCamel(in string) string {
	if IsSnakeCase(in) {
		parts := strings.Split(in, "_")
		var out string
		for _, part := range parts {
			out += ToCamel(part)
		}

		return out
	}

	return in
}

func ToSnakeCase(in string) string {
	runes := []rune(in)
	length := len(runes)

	var out []rune
	for i := 0; i < length; i++ {
		if i > 0 && unicode.IsUpper(runes[i]) && ((i+1 < length && unicode.IsLower(runes[i+1])) || unicode.IsLower(runes[i-1])) {
			out = append(out, '_')
		}
		out = append(out, unicode.ToLower(runes[i]))
	}

	return string(out)
}

func IsSnakeCase(in string) bool {
	if strings.Contains(in, "_") {
		return true
	}

	return false
}

var keywords = map[string]bool{
	"const":       true,
	"var":         true,
	"func":        true,
	"type":        true,
	"import":      true,
	"package":     true,
	"chan":        true,
	"interface":   true,
	"map":         true,
	"struct":      true,
	"break":       true,
	"case":        true,
	"continue":    true,
	"default":     true,
	"else":        true,
	"fallthrough": true,
	"for":         true,
	"goto":        true,
	"if":          true,
	"range":       true,
	"return":      true,
	"select":      true,
	"switch":      true,
	"defer":       true,
	"go":          true,
}

var commonInitialisms = map[string]bool{
	"API":   true,
	"ASCII": true,
	"DNS":   true,
	"ID":    true,
	"IP":    true,
	"JSON":  true,
	"QPS":   true,
	"RAM":   true,
	"RPC":   true,
	"UID":   true,
	"UUID":  true,
	"URI":   true,
	"URL":   true,
	"ERP":   true,
}

func IsBuiltinKeywords(in string) bool {
	if _, ok := keywords[in]; ok {
		return true
	}

	return false
}
