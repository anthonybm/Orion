package windowshelpers

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// getRegistryKey takes in the name of the key and outputs a *registry.Key if found
// Ex: 'HKEY_LOCAL_MACHINE\Software\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\Results\Detect'
func getRegistryKey(k string) (*registry.Key, error) {
	registryMap := map[string]registry.Key{
		"HKEY_CLASSES_ROOT":     registry.CLASSES_ROOT,
		"HKEY_CURRENT_USER":     registry.CURRENT_USER,
		"HKEY_LOCAL_MACHINE":    registry.LOCAL_MACHINE,
		"HKEY_USERS":            registry.USERS,
		"HKEY_CURRENT_CONFIG":   registry.CURRENT_CONFIG,
		"HKEY_PERFORMANCE_DATA": registry.PERFORMANCE_DATA,
	}

	// TODO: Check if valid registry path delimiters for Windows

	// Split key parts
	keyparts := strings.SplitN(k, `\`, 2)
	if len(keyparts) != 2 {
		return nil, fmt.Errorf("failed to parse key '%s' - expected 2 keyparts, got %s", k, len(keyparts))
	}

	rkey, err := registry.OpenKey(registryMap[keyparts[0]], keyparts[1], registry.READ|registry.QUERY_VALUE|registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return nil, fmt.Errorf("failed to open key '%s': %W", k, err)
	}
	return &rkey, nil
}

// GetRegistryValue takes a *registry.Key and string value name and returns the value found for that key if exists
// Ex: 'HKEY_LOCAL_MACHINE\Software\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\Results\Detect', value: 'LastError'
func GetRegistryValue(rKeyName, rValName string) (interface{}, string, string, error) {
	rKey, err := getRegistryKey(rKeyName)

	size, vtype, err := rKey.GetValue(rValName, nil)
	if err != nil {
		return nil, "", "", err
	}

	data := make([]byte, size)
	_, _, err = rKey.GetValue(rValName, data)
	if err != nil {
		return nil, "", "", err
	}

	var res interface{}
	var strres string
	switch vtype {
	case registry.SZ, registry.EXPAND_SZ:
		strres, _, err = rKey.GetStringValue(rValName)
		res = strres
	case registry.NONE, registry.BINARY:
		res, _, err = rKey.GetBinaryValue(rValName)
		strres = fmt.Sprintf("%x", res)
	case registry.QWORD, registry.DWORD:
		res, _, err = rKey.GetIntegerValue(rValName)
		strres = fmt.Sprint("%d", res)
	case registry.MULTI_SZ:
		res, _, err = rKey.GetStringsValue(rValName)
		strres = strings.Join(res.([]string), " ")
	case registry.DWORD_BIG_ENDIAN, registry.LINK, registry.RESOURCE_LIST, registry.FULL_RESOURCE_DESCRIPTOR, registry.RESOURCE_REQUIREMENTS_LIST:
		fallthrough
	default:
		res = data
		strres = fmt.Sprintf("%x", data)
	}
	return res, strres, vtype, err
}
