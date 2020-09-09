package util

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	"go.uber.org/zap"
)

// InterfaceToString uses type assertion to convert input interface to string if possible
func InterfaceToString(i interface{}) (string, error) {
	if i == nil {
		return "nil", nil
	}
	if val, ok := i.(string); ok {
		return val, nil
	}
	if val, ok := i.(int); ok {
		return strconv.Itoa(val), nil
	}
	if val, ok := i.(bool); ok {
		return strconv.FormatBool(val), nil
	}
	if val, ok := i.(float64); ok {
		return strconv.FormatInt(int64(val), 10), nil
	}
	if val, ok := i.(time.Time); ok {
		return val.UTC().Format(time.RFC3339), nil
	}
	if val, ok := i.([]uint8); ok {
		return string(val[:]), nil
		// return fmt.Sprintf("b64:%s", base64.StdEncoding.EncodeToString(val)), nil
	}
	if val, ok := i.([]byte); ok {
		return string(val[:]), nil
	}
	if val, ok := i.([]interface{}); ok {
		res := []string{}
		for _, item := range val {
			itemVal, _ := InterfaceToString(item)
			res = append(res, itemVal)
		}
		return strings.Join(res, ","), nil
	}
	zap.L().Error(fmt.Sprintf("Could not convert [%s] of type [%s] to [string] - no method implemented to handle, please add it", i, reflect.TypeOf(i).String()))
	return "ERROR", fmt.Errorf("Could not convert [%s] of type [%s] to [string] - no method implemented to handle, please add it", i, reflect.TypeOf(i).String())
}

// MapToJSONString converts an input map[string]interface{} to an appropriate json string for output writing
func MapToJSONString(m map[string]interface{}) string {
	pairs := []string{}
	for k, v := range m {
		kVal, err := InterfaceToString(k)
		if err != nil {
			kVal = "ERROR"
			zap.L().Error(err.Error())
		}
		kVal = strings.TrimSpace(kVal)

		vVal, err := InterfaceToString(v)
		if err != nil {
			vVal = "ERROR"
			zap.L().Error(err.Error())
		}
		vVal = strings.TrimSpace(vVal)
		pairs = append(pairs, fmt.Sprintf(`"%s": "%s"`, kVal, vVal))
	}
	res := strings.Join(pairs, ", ")
	return fmt.Sprintf("{ %s }", res)
}

// FileExtension split filepath string to return the file extension
func FileExtension(filepath string) string {
	segments := strings.Split(filepath, ".")
	return segments[len(segments)-1]
}

// Prepend to slice
func Prepend(dest []string, value string) []string {
	if cap(dest) > len(dest) {
		dest = dest[:len(dest)+1]
		copy(dest[1:], dest)
		dest[0] = value
		return dest
	}

	// No room, new slice needs to be allocated
	// Use some extra space for future
	res := make([]string, len(dest)+1, len(dest)+5)
	res[0] = value
	copy(res[1:], dest)
	return res
}

// GetUsernameFromPath returns the last entry of a filepath split by delimiter, typically username for a username path
func GetUsernameFromPath(path string) string {
	userpath := strings.Split(path, "/")
	if strings.Contains(path, "Users") {
		revuserpath := reverseStringSlice(userpath)
		userindex := len(userpath) - 1 - indexInStringSlice(revuserpath, "Users") + 1
		return userpath[userindex]
	} else if strings.Contains(path, "private/var") {
		revuserpath := reverseStringSlice(userpath)
		userindex := len(userpath) - 1 - indexInStringSlice(revuserpath, "var") + 1
		return userpath[userindex]
	}
	return "ERROR"
}

func indexInStringSlice(s []string, substr string) int {
	for i := range s {
		if s[i] == substr {
			return i
		}
	}
	return -1
}

func reverseAny(s interface{}) {
	n := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
}

func reverseStringSlice(s []string) []string {
	a := make([]string, len(s))
	copy(a, s)

	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}

	return a
}

func reverse(s []interface{}) []interface{} {
	a := make([]interface{}, len(s))
	copy(a, s)

	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}

	return a
}

// SliceContainsString outputs true if slice contains str
func SliceContainsString(slice []string, str string) bool {
	for _, item := range slice {
		if str == item {
			return true
		}
		if strings.Contains(item, str) {
			return true
		}
	}
	return false
}

// GetPrintableString returns a string where non-printables in str are omitted
func GetPrintableString(str string) string {
	return strings.Map(func(r rune) rune {
		if !unicode.IsPrint(r) {
			return -1
		}
		return r
	}, strings.TrimSpace(str))
}

// EntryFromMap returns a slice of strings based on input map m and associated with and in order by headers/keys
func EntryFromMap(m map[string]string, headers []string) ([]string, error) {
	if len(headers) == 0 {
		return []string{}, errors.New("EntryFromMap - size of input headers is = 0")
	}
	if len(m) == 0 {
		return []string{}, errors.New("EntryFromMap - size of input map is = 0")
	}
	if len(m) != len(headers) {
		return []string{}, errors.New("EntryFromMap - size of map [" + strconv.Itoa(len(m)) + "] does not match size of headers [" + strconv.Itoa(len(headers)) + "]")
	}
	entry := make([]string, len(headers))
	for i, h := range headers {
		mv := m[h]
		entry[i] = mv
	}
	return entry, nil
}

// UnsafeEntryFromMap returns a slice of strings in order of the header
func UnsafeEntryFromMap(m interface{}, headers []string) ([]string, error) {
	switch u := m.(type) {
	case map[string]string:
		if len(headers) == 0 {
			return []string{}, errors.New("EntryFromMap - size of input headers is = 0")
		}
		entry := make([]string, len(headers))
		for i, h := range headers {
			mv := u[h]
			entry[i] = mv
		}
		return entry, nil
	case map[string]interface{}:
		if len(headers) == 0 {
			return []string{}, errors.New("EntryFromMap - size of input headers is = 0")
		}
		entry := make([]string, len(headers))
		for i, h := range headers {
			mv := u[h]
			val, err := InterfaceToString(mv)
			if err != nil {
				entry[i] = "ERR"
				zap.L().Error(err.Error())
			} else {
				entry[i] = val
			}
		}
		return entry, nil
	}
	return nil, errors.New("unimplemented parsing of" + reflect.TypeOf(m).String())
}

// InitializeMapToEmptyString returns a map that is initialized to empty string for given headers
func InitializeMapToEmptyString(m map[string]string, headers []string) map[string]string {
	for _, header := range headers {
		m[header] = ""
	}
	return m
}

func valueFromNestedMap(obj map[string]interface{}, key string) interface{} {
	if val, ok := obj[key]; ok {
		return val
	}
	for _, v := range obj {
		if _, ok := v.(map[string]interface{}); ok {
			item := valueFromNestedMap(v.(map[string]interface{}), key)
			if item != nil {
				return item
			}
		}
	}
	return nil
}
