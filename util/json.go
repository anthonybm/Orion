package util

import "errors"

// JSONGetValueFromKey takes in an interface representing json data (map[string]interface) and a key, and returns the value for that key if it finds it
func JSONGetValueFromKey(j interface{}, key string) (interface{}, error) {
	if _, ok := j.(map[string]interface{}); !ok {
		return nil, errors.New("input json was not type map[string]interface{}")
	}

	ret := valueFromNestedMap(j.(map[string]interface{}), key)
	return ret, nil
}
