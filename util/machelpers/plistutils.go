// +build darwin

package machelpers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"

	"howett.net/plist"
)

// plistFromBytes decodes a binary or XML based PLIST using github.com/DHowett/go-plist library and returns an interface{} or propagates the error raised by the library
func plistFromBytes(plistBytes []byte) (interface{}, error) {
	var data interface{}
	decoder := plist.NewDecoder(bytes.NewReader(plistBytes))

	err := decoder.Decode(&data)
	if err != nil {
		return data, err
	}
	return data, nil
}

// PlistFromFilepath decodes a binary or XML based Plist using local method
func plistFromFilepath(fp string) (interface{}, error) {
	f, err := ioutil.ReadFile(fp)
	if err != nil {
		return nil, err
	}
	return plistFromBytes(f)
}

// DecodePlistBytes decodes a binary or XML based Plist from bytes into an interface. Returns error from library
func DecodePlistBytes(plistBytes []byte) (interface{}, error) {
	return plistFromBytes(plistBytes)
}

// DecodePlist returns an array of maps corresponding to entries within the given plist, values are interface
func DecodePlist(fp, targetPath string) ([]map[string]interface{}, error) {
	plistInterfaceData, err := plistFromFilepath(filepath.Join(targetPath, fp))
	if err != nil {
		return nil, err
	}
	// fmt.Println(reflect.TypeOf(plistInterfaceData).String())

	if _, ok := plistInterfaceData.([]interface{}); ok {
		var mapslice []map[string]interface{}
		// Assert type of plist as array and iterate over it
		for _, e := range plistInterfaceData.([]interface{}) {
			// type of e should be map[string]interface{}
			mapslice = append(mapslice, e.(map[string]interface{}))
		}

		// For debugging, print the parsed Plist as JSON
		// j, _ := json.MarshalIndent(mapslice, "", "	")
		// fmt.Println(string(j))

		return mapslice, nil
	}
	if _, ok := plistInterfaceData.(map[string]interface{}); ok {
		var slice []map[string]interface{}
		slice = append(slice, plistInterfaceData.(map[string]interface{}))
		return slice, nil
	}
	return nil, errors.New("Failed to decode plist into defined Go Types")
}

// GetSingleValueFromPlist returns the first encountered value from a given plist as a string
func GetSingleValueFromPlist(data []map[string]interface{}, key string) (string, error) {
	// // Try to get top level value
	// for _, item := range data {
	// 	if val, ok := item[key]; ok {
	// 		return fmt.Sprint(val), nil
	// 	}
	// }
	for _, item := range data {
		RET = nil
		KEY = ""
		recurseValueFromNestedMap(item, key)
	}
	if KEY == key {
		return RET.(string), nil
	}

	return "ERROR", errors.New("did not find a value for given key '" + key + "'")
}

// GetSingleValueInterfaceFromPlist returns the first encountered value from a given plist as an interface
func GetSingleValueInterfaceFromPlist(data []map[string]interface{}, key string) (interface{}, error) {
	for _, item := range data {
		RET2 = nil
		KEY2 = ""
		recurseValueFromNestedMap(item, key)
	}
	if KEY == key {
		return RET, nil
	}
	return "ERROR", errors.New("did not find a value for given key '" + key + "'")
}

// RET Global return NOTE: THIS IS VERY MESSY
var RET interface{}
var KEY string
var RET2 interface{}
var KEY2 string

// TODO FIX METHOD OF DOING THIS
// recurseValueFromNestedMap searches for 'key' in the map 'm'
// recurses until it finds a value for 'key' if it exists
func recurseValueFromNestedMap(m map[string]interface{}, key string) interface{} {
	//for _, i := range m {
	for k, v := range m {
		// fmt.Println("Key : ", k)
		if k == key {
			// fmt.Println("Val: ", v)
			RET = v
			KEY = key
		}
		val, ok := v.(map[string]interface{})
		if ok {
			recurseValueFromNestedMap(val, key)
		}

	}
	return nil
}

// isPrimitiveObject determines if given object is of a primative type [uint64, float64, bool, string, []int8]
func isPrimitiveObject(obj interface{}) (interface{}, bool) {
	if v, ok := obj.(uint64); ok {
		return v, ok
	}
	if v, ok := obj.(float64); ok {
		return v, ok
	}
	if v, ok := obj.(bool); ok {
		return v, ok
	}
	if v, ok := obj.(string); ok {
		return v, ok
	}
	if v, ok := obj.([]uint8); ok {
		return v, ok
	}
	return obj, false
}

func PrintPlistAsJSON(obj interface{}) {
	b, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		log.Fatalf("Error while marshalling Json:%s", err)
	}
	fmt.Println(string(b))
}
