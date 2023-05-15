package db2struct

import (
	"fmt"
	"go/format"
	"strconv"
	"strings"
	"unicode"

	"github.com/go-season/ginctl/pkg/util/str"
)

const (
	goByteArray      = "[]byte"
	gureguNullInt    = "null.Int"
	sqlNullInt       = "sql.NullInt64"
	goInt            = "int"
	goInt64          = "int64"
	gureguNullFloat  = "null.Float"
	sqlNullFloat     = "sql.NullFloat64"
	goFloat          = "float"
	goFloat32        = "float32"
	goFloat64        = "float64"
	gureguNullString = "null.String"
	sqlNullString    = "sql.NullString"
	gureguNullTime   = "null.Time"
	goTime           = "time.Time"
	xzTime           = "orm.LocalTime"
)

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
}

var intToWordMap = []string{
	"zero",
	"one",
	"two",
	"three",
	"four",
	"five",
	"six",
	"seven",
	"eight",
	"night",
	"nine",
}

func Generate(columnTypes map[string]map[string]string, columnSorted []string, tableName string, structName string, pkgName string, jsonAnnotation bool, gormAnnotation bool, expectedExtends string) ([]byte, error) {
	var dbTypes string
	dbTypes = generateMysqlTypes(columnTypes, columnSorted, jsonAnnotation, gormAnnotation, expectedExtends)
	src := fmt.Sprintf("type %s %s\n}",
		str.ToCamel(structName),
		dbTypes)
	if gormAnnotation == true {
		tableNameFunc := "// TableName sets insert table name for this struct type\n" +
			"func (" + strings.ToLower(string(structName[0])) + " *" + str.ToCamel(structName) + ") TableName() string {\n" +
			" 		return \"" + tableName + "\"\n" +
			"}"
		src = fmt.Sprintf("%s\n%s", src, tableNameFunc)
	}
	formatted, err := format.Source([]byte(src))
	if err != nil {
		err = fmt.Errorf("error formatting: %s, was formatting\n%s", err, src)
	}
	return formatted, err
}

func fmtFieldName(s string) string {
	name := lintFieldName(s)
	runes := []rune(name)
	for i, c := range runes {
		ok := unicode.IsLetter(c) || unicode.IsDigit(c)
		if i == 0 {
			ok = unicode.IsLetter(c)
		}
		if !ok {
			runes[i] = '_'
		}
	}
	return string(runes)
}

func lintFieldName(name string) string {
	if name == "_" {
		return name
	}

	for len(name) > 0 && name[0:1] == "_" {
		name = name[1:]
	}

	allower := true
	for _, r := range name {
		if !unicode.IsLower(r) {
			allower = false
			break
		}
	}
	if allower {
		runes := []rune(name)
		if u := strings.ToUpper(name); commonInitialisms[u] {
			copy(runes[0:], []rune(u))
		} else {
			runes[0] = unicode.ToUpper(runes[0])
		}
		return string(runes)
	}

	runes := []rune(name)
	w, i := 0, 0
	for i+1 <= len(runes) {
		eow := false

		if i+1 == len(runes) {
			eow = true
		} else if runes[i+1] == '_' {
			eow = true
			n := 1
			for i+n+1 < len(runes) && runes[i+n+1] == '_' {
				n++
			}

			if i+n+1 < len(runes) && unicode.IsDigit(runes[i]) && unicode.IsDigit(runes[i+n+1]) {
				n--
			}

			copy(runes[i+1:], runes[i+n+1:])
			runes = runes[:len(runes)-n]
		} else if unicode.IsLower(runes[i]) && !unicode.IsLower(runes[i+1]) {
			eow = true
		}
		i++
		if !eow {
			continue
		}

		word := string(runes[w:i])
		if u := strings.ToUpper(word); commonInitialisms[u] {
			copy(runes[w:], []rune(u))
		} else if strings.ToLower(word) == word {
			runes[w] = unicode.ToUpper(runes[w])
		}
		w = i
	}
	return string(runes)
}

func stringifyFirstChar(str string) string {
	first := str[:1]

	i, err := strconv.ParseInt(first, 10, 8)
	if err != nil {
		return str
	}

	return intToWordMap[i] + "_" + str[1:]
}
