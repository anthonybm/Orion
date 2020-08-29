// +build darwin

package machelpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"

	"go.uber.org/zap"
	plist "howett.net/plist"
)

/*
Inspired by https://github.com/danielpaulus/nskeyedarchiver and by CrowdStrike's automactc (and its inspirations)
Some modifications were made to the original code
Cloned to this library to avoid third-party dependence when building
*/

/* CONSTANTS */
const (
	archiverKey     = "$archiver"
	nsKeyedArchiver = "NSKeyedArchiver"
	versionKey      = "$version"
	topKey          = "$top"
	objectsKey      = "$objects"
	nsObjects       = "NS.objects"
	nsKeys          = "NS.keys"
	class           = "$class"
	className       = "$classname"
	versionValue    = 100000
)

const (
	nsArray        = "NSArray"
	nsMutableArray = "NSMutableArray"
	nsSet          = "NSSet"
	nsMutableSet   = "NSMutableSet"
)

const (
	nsDictionary        = "NSDictionary"
	nsMutableDictionary = "NSMutableDictionary"
)

// NS primatives
const (
	nsMutableString = "NSMutableString"
	nsMutableData   = "NSMutableData"
)

/* CONSTANTS END */

// UnarchiveNSKeyedArchiver extracts an NSKeyedArchiver Plist from a given filepath, (XML or Binary), and returns an array of the NSObjects converted to usable Go Types
// Primitives will be extracted just like regular Plist primitives (string, float64, int64, []uint8 etc.).
// NSArray, NSMutableArray, NSSet and NSMutableSet will transformed into []interface{}
// NSDictionary and NSMutableDictionary will be transformed into map[string] interface{}
func UnarchiveNSKeyedArchiver(fp string) ([]interface{}, error) {
	plistData, err := plistFromFilepath(fp)
	if err != nil {
		return nil, fmt.Errorf("Unarchive NSKeyedArchiver: %s", err.Error())
	}
	nsKeyedArchiverData := plistData.(map[string]interface{})

	err = verifyCorrectArchiver(nsKeyedArchiverData)
	if err != nil {
		return nil, fmt.Errorf("Unarchive NSKeyedArchiver: %s", err.Error())
	}
	return extractNSObjectsFromTop(nsKeyedArchiverData[topKey].(map[string]interface{}), nsKeyedArchiverData[objectsKey].([]interface{}))
}

func UnarchiveNSKeyedArchiverBytes(f []byte) ([]interface{}, error) {
	plistData, err := plistFromBytes(f)
	if err != nil {
		return nil, fmt.Errorf("Unarchive NSKeyedArchiver: %s", err.Error())
	}
	nsKeyedArchiverData := plistData.(map[string]interface{})
	// fmt.Println(nsKeyedArchiverData)

	ret, err := extractNSObjectsFromTop(nsKeyedArchiverData[topKey].(map[string]interface{}), nsKeyedArchiverData[objectsKey].([]interface{}))
	if err != nil {
		zap.L().Error(err.Error(), zap.String("util", "nskeyedarchiver"))
	}
	return ret, err
}

func extractNSObjectsFromTop(top map[string]interface{}, objects []interface{}) ([]interface{}, error) {
	// fmt.Println("TOP", top)
	// fmt.Println("OBJ", objects)
	objectCount := len(top)
	if root, ok := top["root"]; ok {
		return extractNSObjects([]plist.UID{root.(plist.UID)}, objects)
	}
	objectRefs := make([]plist.UID, objectCount)

	// convert the Dictionary with the objectReferences into a flat list of UIDs, so we can reuse the extractNSObjects function later
	for i := 0; i < objectCount; i++ {
		// objectIndex := top[fmt.Sprintf("%d", i)].(plist.UID)
		objectIndex := getReverseKey(top, uint64(i))
		objectRefs[i] = objectIndex
	}
	return extractNSObjects(objectRefs, objects)
}

func getReverseKey(top map[string]interface{}, key uint64) plist.UID {
	for _, v := range top {
		if v == key {
			switch u := v.(type) {
			case plist.UID:
				return u
			case uint64:
				return plist.UID(u)
			default:
				zap.L().Error("Could not parse key - "+reflect.TypeOf(u).String(), zap.String("util", "nskeyedarchiver"))
			}
		}
	}
	return plist.UID(0)
}

func extractNSObjects(objectRefs []plist.UID, objects []interface{}) ([]interface{}, error) {
	objectCount := len(objectRefs)
	returnValue := make([]interface{}, objectCount)
	for i := 0; i < objectCount; i++ {
		objectIndex := objectRefs[i]
		objectRef := objects[objectIndex]
		if object, ok := isPrimitiveObject(objectRef); ok {
			returnValue[i] = object
			continue
		}
		if object, ok := isArrayObject(objectRef.(map[string]interface{}), objects); ok {
			extractNSObjects, err := extractNSObjects(toUidList(object[nsObjects].([]interface{})), objects)
			if err != nil {
				return nil, err
			}
			returnValue[i] = extractNSObjects
			continue
		}

		if object, ok := isDictionaryObject(objectRef.(map[string]interface{}), objects); ok {
			dictionary, err := extractDictionary(object, objects)
			if err != nil {
				return nil, err
			}
			returnValue[i] = dictionary
			continue
		}

		if object, ok := isNSPrimativeObject(objectRef.(map[string]interface{}), objects); ok {
			for k, v := range object {
				if strings.HasPrefix(k, "NS") {
					returnValue[i] = v
					break
				}
			}
			// fmt.Println(objectRef)
			continue
		}

		objectType := reflect.TypeOf(objectRef).String()
		// fmt.Println(objectRef)
		// panic(fmt.Sprintf("Unknown object type:%s", objectType))
		return nil, fmt.Errorf("Unknown object type:%s for objects:%s", objectType, objects)

	}
	return returnValue, nil
}

func isNSPrimativeObject(object map[string]interface{}, objects []interface{}) (map[string]interface{}, bool) {
	className, err := resolveClass(object[class], objects)
	if err != nil {
		zap.L().Error(fmt.Sprintf("could not get classname for %s", object[class]), zap.String("util", "nskeyedarchiver"))
		return nil, false
	}
	if className == nsMutableString || className == nsMutableData {
		return object, true
	}
	zap.L().Error(fmt.Sprintf("could not use classname %s", className), zap.String("util", "nskeyedarchiver"))
	return object, false
}

func isArrayObject(object map[string]interface{}, objects []interface{}) (map[string]interface{}, bool) {
	className, err := resolveClass(object[class], objects)
	if err != nil {
		zap.L().Error(fmt.Sprintf("could not get classname for %s", object[class]), zap.String("util", "nskeyedarchiver"))
		return nil, false
	}
	if className == nsArray || className == nsMutableArray || className == nsSet || className == nsMutableSet {
		return object, true
	}
	// zap.L().Debug(fmt.Sprintf("could not use classname %s as arrayobject", className), zap.String("util", "nskeyedarchiver"))
	return object, false
}

func isDictionaryObject(object map[string]interface{}, objects []interface{}) (map[string]interface{}, bool) {
	className, err := resolveClass(object[class], objects)
	if err != nil {
		zap.L().Error(fmt.Sprintf("could not get classname for %s", object[class]), zap.String("util", "nskeyedarchiver"))
		return nil, false
	}
	if className == nsDictionary || className == nsMutableDictionary {
		return object, true
	}
	// zap.L().Debug(fmt.Sprintf("could not use classname %s as dictionary object", className), zap.String("util", "nskeyedarchiver"))
	return object, false
}

func extractDictionary(object map[string]interface{}, objects []interface{}) (map[string]interface{}, error) {
	keyRefs := toUidList(object[nsKeys].([]interface{}))
	keys, err := extractNSObjects(keyRefs, objects)
	if err != nil {
		return nil, err
	}

	valueRefs := toUidList(object[nsObjects].([]interface{}))
	values, err := extractNSObjects(valueRefs, objects)
	if err != nil {
		return nil, err
	}
	mapSize := len(keys)
	result := make(map[string]interface{}, mapSize)
	for i := 0; i < mapSize; i++ {
		result[keys[i].(string)] = values[i]
	}

	return result, nil
}

func extractNSPrimativeObject(object map[string]interface{}, objects []interface{}) (map[string]interface{}, error) {
	var keyRefs = []plist.UID{}
	keyRefs = append(keyRefs, object[nsKeys].(plist.UID))
	keys, err := extractNSObjects(keyRefs, objects)
	if err != nil {
		return nil, err
	}

	valueRefs := toUidList(object[nsObjects].([]interface{}))
	values, err := extractNSObjects(valueRefs, objects)
	if err != nil {
		return nil, err
	}
	mapSize := len(keys)
	result := make(map[string]interface{}, mapSize)
	for i := 0; i < mapSize; i++ {
		result[keys[i].(string)] = values[i]
	}

	return result, nil
}

func resolveClass(classInfo interface{}, objects []interface{}) (string, error) {
	if v, ok := classInfo.(plist.UID); ok {
		classDict := objects[v].(map[string]interface{})
		return classDict[className].(string), nil
	}
	return "", fmt.Errorf("Could not find class for %s", classInfo)
}

/* INTERNAL */
//toUidList type asserts a []interface{} to a []plist.UID by iterating through the list.
func toUidList(list []interface{}) []plist.UID {
	l := len(list)
	result := make([]plist.UID, l)
	for i := 0; i < l; i++ {
		result[i] = list[i].(plist.UID)
	}
	return result
}

//ToPlist converts a given struct to a Plist using the
//github.com/DHowett/go-plist library. Make sure your struct is exported.
//It returns a string containing the plist.
func ToPlist(data interface{}) string {
	buf := &bytes.Buffer{}
	encoder := plist.NewEncoder(buf)
	encoder.Encode(data)
	return buf.String()
}

//Print an object as JSON for debugging purposes, careful log.Fatals on error
func printAsJSON(obj interface{}) {
	b, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		log.Fatalf("Error while marshalling Json:%s", err)
	}
	fmt.Print(string(b))
}

//verifyCorrectArchiver makes sure the nsKeyedArchived plist has all the right keys and values and returns an error otherwise
func verifyCorrectArchiver(nsKeyedArchiverData map[string]interface{}) error {
	if val, ok := nsKeyedArchiverData[archiverKey]; !ok {
		return fmt.Errorf("Invalid NSKeyedAchiverObject, missing key '%s'", archiverKey)
	} else {
		if stringValue := val.(string); stringValue != nsKeyedArchiver {
			return fmt.Errorf("Invalid value: %s for key '%s', expected: '%s'", stringValue, archiverKey, nsKeyedArchiver)
		}
	}
	if _, ok := nsKeyedArchiverData[topKey]; !ok {
		return fmt.Errorf("Invalid NSKeyedAchiverObject, missing key '%s'", topKey)
	}

	if _, ok := nsKeyedArchiverData[objectsKey]; !ok {
		return fmt.Errorf("Invalid NSKeyedAchiverObject, missing key '%s'", objectsKey)
	}

	if val, ok := nsKeyedArchiverData[versionKey]; !ok {
		return fmt.Errorf("Invalid NSKeyedAchiverObject, missing key '%s'", versionKey)
	} else {
		if stringValue := val.(uint64); stringValue != versionValue {
			return fmt.Errorf("Invalid value: %d for key '%s', expected: '%d'", stringValue, versionKey, versionValue)
		}
	}

	return nil
}
