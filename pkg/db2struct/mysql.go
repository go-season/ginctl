package db2struct

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-season/ginctl/pkg/util/str"
	"github.com/go-sql-driver/mysql"
)

var (
	excludeMapField = map[string]bool{
		"id":         true,
		"created_at": true,
		"updated_at": true,
		"deleted_at": true,
	}
)

func GetColumnsFromMysqlTable(dsn, table string) (*map[string]map[string]string, []string, error) {
	var err error
	var db *sql.DB

	db, err = sql.Open("mysql", dsn)
	defer db.Close()

	if err != nil {
		fmt.Println("Error opening mysql db: ", err.Error())
		return nil, nil, err
	}

	columnNameSorted := []string{}

	columnDataTypes := make(map[string]map[string]string)
	columnDataTypeQuery := "SELECT COLUMN_NAME, COLUMN_KEY, DATA_TYPE, IS_NULLABLE, COLUMN_COMMENT FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = ? AND table_name = ? order by ordinal_position asc"

	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		fmt.Println("Error parse conn dsn: ", err.Error())
		return nil, nil, err
	}

	rows, err := db.Query(columnDataTypeQuery, cfg.DBName, table)
	if err != nil {
		fmt.Println("Error selecting from db: " + err.Error())
		return nil, nil, err
	}
	if rows != nil {
		defer rows.Close()
	} else {
		return nil, nil, errors.New("no results returned for table")
	}

	for rows.Next() {
		var column string
		var columnKey string
		var dataType string
		var nullable string
		var comment string
		rows.Scan(&column, &columnKey, &dataType, &nullable, &comment)

		columnDataTypes[column] = map[string]string{"value": dataType, "nullable": nullable, "primary": columnKey, "comment": comment}
		columnNameSorted = append(columnNameSorted, column)
	}

	return &columnDataTypes, columnNameSorted, err
}

func GenerateReqAndRespTypes(obj map[string]map[string]string, columnSorted []string, jsonTag bool, formTag bool) string {
	structure := ""

	for _, key := range columnSorted {
		if _, ok := excludeMapField[key]; ok {
			continue
		}

		mysqlType := obj[key]

		var valueType string
		valueType = mysqlTypeToGoType(mysqlType["value"], false)
		if valueType == goTime {
			valueType = "string"
		}

		fieldName := fmtFieldName(stringifyFirstChar(key))
		var annotations []string
		if formTag == true {
			annotations = append(annotations, fmt.Sprintf("form:\"%s\"", str.ToLowerCamelCase(fieldName)))
		}
		if jsonTag == true {
			annotations = append(annotations, fmt.Sprintf("json:\"%s\"", str.ToLowerCamelCase(fieldName)))
		}

		if len(annotations) > 0 {
			comment := mysqlType["comment"]
			structure += fmt.Sprintf("	%s %s `%s` // %s\n", fieldName, valueType, strings.Join(annotations, " "), comment)
		} else {
			structure += fmt.Sprintf("	%s %s\n", fieldName, valueType)
		}
	}

	return structure
}

var modelExtends = map[string]string{
	"default":    "orm.Model",
	"ormV2":      "orm.Base",
	"softDelete": "orm.SoftDelete",
	"noORM":      "",
}

func generateMysqlTypes(obj map[string]map[string]string, columnSorted []string, jsonAnnotation bool, gormAnnotation bool, expectedExtends string) string {
	structure := "struct {"
	if extend, ok := modelExtends[expectedExtends]; ok && extend != "" {
		structure += fmt.Sprintf("\n%s\n", extend)
	}

	for _, key := range columnSorted {
		if _, ok := excludeMapField[key]; ok {
			if expectedExtends == "noORM" && key == "id" {
				// nothing doing.
			} else {
				continue
			}
		}

		mysqlType := obj[key]
		nullable := false
		if mysqlType["nullable"] == "YES" {
			nullable = true
		}

		primary := ""
		if mysqlType["primary"] == "PRI" {
			primary = ";primary_key"
		}

		var valueType string
		valueType = mysqlTypeToGoType(mysqlType["value"], nullable)

		fieldName := fmtFieldName(stringifyFirstChar(key))
		var annotations []string
		if gormAnnotation == true {
			annotations = append(annotations, fmt.Sprintf("gorm:\"column:%s%s\"", key, primary))
		}
		if jsonAnnotation == true {
			annotations = append(annotations, fmt.Sprintf("json:\"%s\"", str.ToLowerCamelCase(fieldName)))
		}

		if len(annotations) > 0 {
			comment := mysqlType["comment"]
			comment = strings.Replace(comment, "{", "", -1)
			comment = strings.Replace(comment, "}", "", -1)
			structure += fmt.Sprintf("\n%s %s `%s` // %s", fieldName, valueType, strings.Join(annotations, " "), comment)
		} else {
			structure += fmt.Sprintf("\n%s %s", fieldName, valueType)
		}
	}
	return structure
}

func mysqlTypeToGoType(mysqlType string, nullable bool) string {
	switch mysqlType {
	case "tinyint", "int", "smallint", "mediumint":
		//if nullable {
		//	return sqlNullInt
		//}
		return goInt
	case "bigint":
		//if nullable {
		//	return sqlNullInt
		//}
		return goInt64
	case "char", "enum", "varchar", "longtext", "mediumtext", "text", "tinytext", "json":
		//if nullable {
		//	return sqlNullString
		//}
		return "string"
	case "date", "datetime", "time", "timestamp":
		return xzTime
	case "decimal", "double":
		//if nullable {
		//	return sqlNullFloat
		//}
		return goFloat64
	case "float":
		//if nullable {
		//	return sqlNullFloat
		//}
		return goFloat32
	case "binary", "blob", "longblob", "mediumblob", "varbinary":
		return goByteArray
	}
	return ""
}
