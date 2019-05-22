package migrations

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
)

func bindata_read(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	return buf.Bytes(), nil
}

var __0001_add_messages_contacts_down_db_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\x28\x2d\x4e\x2d\x8a\xcf\x4d\x2d\x2e\x4e\x4c\x4f\x2d\xb6\xe6\x42\x97\x49\xce\xcf\x2b\x49\x4c\x2e\x29\xb6\xe6\x02\x04\x00\x00\xff\xff\xe3\x7e\xc7\x78\x34\x00\x00\x00")

func _0001_add_messages_contacts_down_db_sql() ([]byte, error) {
	return bindata_read(
		__0001_add_messages_contacts_down_db_sql,
		"0001_add_messages_contacts.down.db.sql",
	)
}

var __0001_add_messages_contacts_up_db_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x74\x90\xcd\x4e\xc3\x30\x10\x84\xef\x7e\x8a\x3d\x16\x29\x6f\xd0\x53\xd2\x1a\xba\x22\xd8\xe0\x3a\x34\x3d\x45\xc6\xb5\xc0\x6a\xf3\x23\xbc\x95\xe8\xdb\x23\x2c\x87\x84\xaa\x5c\x67\x77\x67\x67\xbe\x95\xe2\xb9\xe6\xa0\xf3\xa2\xe4\x80\xf7\x20\xa4\x06\x5e\xe3\x56\x6f\xe1\x1c\xdc\x67\xd3\xba\x10\xcc\xbb\x0b\xb0\x60\xfe\x00\x45\x29\x0b\xa8\x04\xbe\x54\x3c\x6e\x8a\xaa\x2c\x33\x66\xfb\x8e\x8c\xa5\xc6\x1f\xe0\x35\x57\xab\x4d\xae\xae\x86\xae\xa3\x86\x2e\x83\x1b\xc7\x19\x4b\xb6\x57\x2a\xb9\x2f\x02\xcd\x6b\x9d\x31\x7b\xea\xed\x11\x0a\x7c\x40\xa1\x33\x46\xbe\x75\x81\x4c\x3b\xfc\x2a\xa3\xad\xfd\x30\xf1\x71\xba\x1a\x9f\x4d\x46\xc3\xf9\xed\xe4\x6d\x73\x74\x97\x98\x9e\xdd\x2d\x19\x4b\xa5\x51\xac\x79\x0d\x53\xfa\x00\x52\xfc\x6d\xbd\x98\x86\xb3\xbb\x7f\x61\xa5\xed\x04\x6b\x64\xf1\xac\xf0\x29\x57\x7b\x78\xe4\xfb\x19\x97\xce\xb4\xee\x06\x2e\xea\x07\x6f\x63\xf4\xb9\xf8\x43\x09\xc5\x5c\x0a\x64\x28\x6a\x37\x1a\xc2\x0e\xf5\x46\x56\x1a\x94\xdc\xe1\x7a\xc9\xbe\x03\x00\x00\xff\xff\x70\xc1\xe7\x30\xe4\x01\x00\x00")

func _0001_add_messages_contacts_up_db_sql() ([]byte, error) {
	return bindata_read(
		__0001_add_messages_contacts_up_db_sql,
		"0001_add_messages_contacts.up.db.sql",
	)
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		return f()
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() ([]byte, error){
	"0001_add_messages_contacts.down.db.sql": _0001_add_messages_contacts_down_db_sql,
	"0001_add_messages_contacts.up.db.sql": _0001_add_messages_contacts_up_db_sql,
}
// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for name := range node.Children {
		rv = append(rv, name)
	}
	return rv, nil
}

type _bintree_t struct {
	Func func() ([]byte, error)
	Children map[string]*_bintree_t
}
var _bintree = &_bintree_t{nil, map[string]*_bintree_t{
	"0001_add_messages_contacts.down.db.sql": &_bintree_t{_0001_add_messages_contacts_down_db_sql, map[string]*_bintree_t{
	}},
	"0001_add_messages_contacts.up.db.sql": &_bintree_t{_0001_add_messages_contacts_up_db_sql, map[string]*_bintree_t{
	}},
}}
