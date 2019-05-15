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

var __0001_add_messages_contacts_up_db_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x74\x90\xcd\x4e\xc3\x30\x10\x84\xef\x7e\x8a\x3d\x16\x29\x6f\xd0\x53\xd2\x1a\xba\x22\xd8\xe0\x3a\x34\x3d\x45\xc6\xac\x20\x6a\xf3\x23\xbc\x95\xe8\xdb\x23\xa2\x84\xb8\x55\xb9\xce\xce\xce\xee\x7c\x2b\x23\x53\x2b\xc1\xa6\x59\x2e\x01\xef\x41\x69\x0b\xb2\xc4\xad\xdd\xc2\x29\xd0\x57\xd5\x50\x08\xee\x83\x02\x2c\x44\xfd\x0e\xaf\xa9\x59\x6d\x52\x03\x85\xc2\x97\x42\x0e\x66\x55\xe4\x79\x22\x7c\xd7\xb2\xf3\x5c\x45\x9e\xcb\x21\xb5\x5c\xf1\xb9\xa7\x69\x9c\x88\x31\xf9\x4a\x65\xfa\x66\xb0\xb2\xb4\x89\xf0\xc7\xce\x1f\x20\xc3\x07\x54\x36\x11\x5c\x37\x14\xd8\x35\xfd\x9f\x32\xc5\xfa\x4f\x37\x1c\x1e\xb7\xa6\x63\x73\x50\x7f\x7a\x3b\xd6\xbe\x3a\xd0\x19\xb2\x5c\x67\xe2\x6e\x29\xc4\xd8\x1b\xd5\x5a\x96\x30\x7f\x1f\x40\xab\xcb\xe2\x8b\x79\x18\xed\xfd\xcb\x6b\x74\x5f\xf1\x7a\x36\xf8\x94\x9a\x3d\x3c\xca\x7d\xc4\xa5\x75\x0d\xdd\xc0\xc5\x5d\x5f\xfb\xe1\xf5\x58\xfc\xa5\x84\x2a\x96\x02\x3b\x1e\xb4\x1b\x0d\x61\x87\x76\xa3\x0b\x0b\x46\xef\x70\xbd\x14\x3f\x01\x00\x00\xff\xff\xb4\xd5\xa9\x88\xe7\x01\x00\x00")

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
