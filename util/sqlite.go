package util

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"

	// using sqlite implementation
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

var (
	sqlDriverName = "sqlite3"
)

// DBColumnNames returns a slice of strings representing column headers for input tablename of dbfile
func DBColumnNames(dbfile, tablename string) ([]string, error) {
	q := fmt.Sprintf("SELECT * FROM %s", tablename)
	db, err := sql.Open(sqlDriverName, dbfile)
	defer db.Close()
	if err != nil {
		return []string{}, errors.New("Failed to open '" + dbfile + "': " + err.Error())
	}
	zap.L().Debug(fmt.Sprintf("Opened '%s', will try to get column names via query '%s'", dbfile, q))

	res, err := db.Query(q)
	if err != nil {
		return []string{}, errors.New("Failed to query '" + q + "' on database '" + dbfile + "': " + err.Error())
	}
	defer res.Close() // defer after o.w. panic on nil dereference
	colnames, err := res.Columns()
	if err != nil {
		return []string{}, errors.New("Failed to get columns of '" + dbfile + "': " + err.Error())
	}
	return colnames, nil
}

func UnsafeQueryDBToMap(dbfile, query string) ([]map[string]interface{}, error) {
	// Open SQLite file
	db, err := sql.Open(sqlDriverName, dbfile)
	defer db.Close()
	if err != nil {
		return []map[string]interface{}{}, errors.New("Failed to open '" + dbfile + "': " + err.Error())
	}
	zap.L().Debug(fmt.Sprintf("Opened '%s', will try to query '%s'", dbfile, query))

	// Send query to db
	rows, err := db.Query(query)
	if err != nil {
		return []map[string]interface{}{}, errors.New("Failed to query '" + query + "' on database '" + dbfile + "': " + err.Error())
	}
	defer rows.Close() // defer after o.w. panic on nil dereference

	colnames, err := rows.Columns()
	if err != nil {
		return []map[string]interface{}{}, errors.New("Failed to get columns of '" + dbfile + "': " + err.Error())
	}

	// Scan needs an array of pointers to the values it is setting
	// This creates the object and sets the values correctly
	cols := make([]interface{}, len(colnames))
	colPtrs := make([]interface{}, len(colnames))
	for i := 0; i < len(colnames); i++ {
		colPtrs[i] = &cols[i]
	}

	var valmap = make(map[string]interface{})
	var colSet = make(map[string]bool)                 // track keys you find
	var entrySlice = make([]map[string]interface{}, 0) // save entry valmaps
	for rows.Next() {
		err := rows.Scan(colPtrs...)
		if err != nil {
			return []map[string]interface{}{}, err
		}
		for i, col := range cols {
			valmap[colnames[i]] = col
		}

		var entry = make(map[string]interface{})
		for k, v := range valmap {
			colSet[k] = true // mark you've seen this column
			switch u := v.(type) {
			case string:
				entry[k] = u
			case float64:
				entry[k] = strconv.FormatFloat(u, 'E', -1, 64)
			case int64:
				entry[k] = strconv.FormatInt(u, 10)
			case []uint8:
				entry[k] = fmt.Sprintf("b64:%s", base64.StdEncoding.EncodeToString(u))
			default:
				zap.L().Error("Type <" + reflect.TypeOf(v).String() + "> not currently processed by sqlite util!!")
				entry[k] = "ERR-UNIMPL"
			}
		}
		entrySlice = append(entrySlice, entry)
	}
	err = rows.Err()
	if err != nil {
		return []map[string]interface{}{}, err
	}

	return entrySlice, nil
}

func UnsafeQueryDB(dbfile, query string) ([][]string, error) {
	values := [][]string{}

	// Open SQLite file
	db, err := sql.Open(sqlDriverName, dbfile)
	defer db.Close()
	if err != nil {
		return [][]string{}, errors.New("Failed to open '" + dbfile + "': " + err.Error())
	}
	zap.L().Debug(fmt.Sprintf("Opened '%s', will try to query '%s'", dbfile, query))

	// Send query to db
	rows, err := db.Query(query)
	if err != nil {
		return [][]string{}, errors.New("Failed to query '" + query + "' on database '" + dbfile + "': " + err.Error())
	}
	defer rows.Close() // defer after o.w. panic on nil dereference

	colnames, err := rows.Columns()
	if err != nil {
		return [][]string{}, errors.New("Failed to get columns of '" + dbfile + "': " + err.Error())
	}

	// Scan needs an array of pointers to the values it is setting
	// This creates the object and sets the values correctly
	cols := make([]interface{}, len(colnames))
	colPtrs := make([]interface{}, len(colnames))
	for i := 0; i < len(colnames); i++ {
		colPtrs[i] = &cols[i]
	}

	var valmap = make(map[string]interface{})
	var colSet = make(map[string]bool)                 // track keys you find
	var entrySlice = make([]map[string]interface{}, 0) // save entry valmaps
	for rows.Next() {
		err := rows.Scan(colPtrs...)
		if err != nil {
			return [][]string{}, err
		}
		for i, col := range cols {
			valmap[colnames[i]] = col
		}

		var entry = make(map[string]interface{})
		for k, v := range valmap {
			colSet[k] = true // mark you've seen this column
			switch u := v.(type) {
			case string:
				entry[k] = u
			case float64:
				entry[k] = strconv.FormatFloat(u, 'E', -1, 64)
			case int64:
				entry[k] = strconv.FormatInt(u, 10)
			case []uint8:
				entry[k] = fmt.Sprintf("b64:%s", base64.StdEncoding.EncodeToString(u))
			default:
				zap.L().Error("Type <" + reflect.TypeOf(v).String() + "> not currently processed by sqlite util!!")
				entry[k] = "ERR-UNIMPL"
			}
		}
		entrySlice = append(entrySlice, entry)
	}
	err = rows.Err()
	if err != nil {
		return [][]string{}, err
	}

	// res := []string{}
	kNames := []string{}
	for k := range colSet {
		kNames = append(kNames, k)
	}
	sort.Strings(kNames)
	for _, m := range entrySlice {
		entry, err := UnsafeEntryFromMap(m, kNames)
		if err != nil {
			zap.L().Error(err.Error())
		}
		values = append(values, entry)
	}

	return values, nil
}

// QueryDB returns an array of entries(string array) ordered by queryHeaders
func QueryDB(dbfile string, query string, queryHeaders []string, forensic bool) ([][]string, error) {
	if !forensic {
		// Open SQLite file
		db, err := sql.Open(sqlDriverName, dbfile)
		defer db.Close()
		if err != nil {
			return [][]string{}, errors.New("Failed to open '" + dbfile + "': " + err.Error())
		}
		zap.L().Debug(fmt.Sprintf("Opened '%s', will try to query '%s'", dbfile, query))

		// Send query to DB
		rows, err := db.Query(query)
		if err != nil {
			return [][]string{}, errors.New("Failed to query '" + query + "' on database '" + dbfile + "': " + err.Error())
		}
		defer rows.Close() // defer after o.w. panic on nil dereference

		var valmap = make(map[string]interface{})
		colnames, err := rows.Columns()
		if err != nil {
			return [][]string{}, errors.New("Failed to get columns of '" + dbfile + "': " + err.Error())
		}

		// Scan needs an array of pointers to the values it is setting
		// This creates the object and sets the values correctly
		cols := make([]interface{}, len(colnames))
		colPtrs := make([]interface{}, len(colnames))
		for i := 0; i < len(colnames); i++ {
			colPtrs[i] = &cols[i]
		}

		res := [][]string{}
		for rows.Next() {
			err := rows.Scan(colPtrs...)
			if err != nil {
				return [][]string{}, err
			}
			for i, col := range cols {
				valmap[colnames[i]] = col
			}

			var headermap = make(map[string]string)
			for i := 0; i < len(queryHeaders); i++ {
				headermap[queryHeaders[i]] = ""
			}

			entry := []string{}
			for k, v := range valmap {
				switch u := v.(type) {
				case string:
					headermap[k] = u
				case float64:
					headermap[k] = strconv.FormatFloat(u, 'E', -1, 64)
				case int64:
					headermap[k] = strconv.FormatInt(u, 10)
				case []uint8:
					headermap[k] = fmt.Sprintf("b64:%s", base64.StdEncoding.EncodeToString(u))
				default:
					zap.L().Error("Type <" + reflect.TypeOf(v).String() + "> not currently processed by sqlite util!!")
				}
			}
			for header := range queryHeaders {
				entry = append(entry, headermap[queryHeaders[header]])
			}

			res = append(res, entry)
		}
		err = rows.Err()
		if err != nil {
			return [][]string{}, err
		}

		return res, nil
	}
	// Forensic mode should copy the file safely to temp location to avoid modifying data
	return [][]string{}, errors.New("Unimplemented Forensic Mode for SQLite")
}
