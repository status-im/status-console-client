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

var __0001_add_messages_contacts_down_db_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\x28\x2d\x4e\x2d\x8a\xcf\x4d\x2d\x2e\x4e\x4c\x4f\x2d\xb6\xe6\x42\x97\x49\xce\xcf\x2b\x49\x4c\x2e\x41\x95\xc9\xc8\x2c\x2e\xc9\x2f\xaa\x8c\x47\x56\x11\x5f\x92\x5f\x90\x99\x6c\xcd\x05\x08\x00\x00\xff\xff\x1b\x57\x14\x62\x5b\x00\x00\x00")

func _0001_add_messages_contacts_down_db_sql() ([]byte, error) {
	return bindata_read(
		__0001_add_messages_contacts_down_db_sql,
		"0001_add_messages_contacts.down.db.sql",
	)
}

var __0001_add_messages_contacts_up_db_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x7c\x92\xcf\x6e\xb3\x30\x10\xc4\xef\x7e\x8a\x3d\xf2\x49\x1c\xbe\x7b\x4f\x06\x96\xc4\x2a\xb5\x5b\xc7\x34\xc9\x09\x51\xc7\x6a\x50\xc2\x1f\xc5\x8e\x54\xde\xbe\x82\x42\xe3\xa6\x69\xae\x33\xf6\x7a\xe6\xb7\x8e\x25\x52\x85\xa0\x68\x94\x21\xb0\x14\xb8\x50\x80\x1b\xb6\x52\x2b\x38\x5b\x73\x2a\x6a\x63\x6d\xf9\x6e\x2c\x04\xa4\xda\x41\x94\x89\x08\x72\xce\x5e\x72\x1c\x4f\xf2\x3c\xcb\x42\xa2\xdb\xc6\x95\xda\x15\xd5\x0e\x5e\xa9\x8c\x97\x54\x5e\x99\xa6\x71\x85\xeb\x3b\x33\xdb\x21\x99\xc6\x5e\xa9\xce\x7c\x38\x50\xb8\x51\x21\xd1\xc7\x56\x1f\x20\x62\x0b\xc6\x55\x48\x5c\x55\x1b\xeb\xca\xba\xfb\x56\xe6\xb1\x7a\x5f\x8e\x0f\x4f\xb7\xe6\xc7\x2e\x83\xba\xf3\xdb\xb1\xd2\xc5\xc1\xf4\x63\x7a\xf2\xef\x81\x90\xa9\x34\xe3\x09\x6e\xe0\x92\xde\x82\xe0\x3f\x5b\x07\x17\xd3\xbb\xf7\x27\xac\xe9\xf4\x04\x6b\x66\xf1\x2c\xd9\x13\x95\x5b\x78\xc4\xad\xc7\xa5\x29\x6b\x73\x03\x97\x6b\xbb\x4a\x8f\xd1\x7d\x71\xa0\xc4\xb8\x2f\x59\x57\xba\x51\xbb\xd1\x10\xd6\x4c\x2d\x45\xae\x40\x8a\x35\x4b\xee\xe7\xde\x57\xd6\xb5\xa7\xbe\xf0\xf3\x17\x5f\x21\x02\x62\xfb\x46\x9b\xdd\xc4\x1c\x12\x4c\x69\x9e\x29\xf8\x7f\x7f\xf5\xbf\xbe\x47\x2a\x24\xb2\x05\x1f\xfa\xfb\x3c\x41\x62\x8a\x12\x79\x8c\x57\xf4\x82\xc1\x14\x1c\x12\xcc\x50\x21\xc4\x74\x15\xd3\x04\x87\xc5\x7d\x06\x00\x00\xff\xff\x28\x6d\x42\x61\xad\x02\x00\x00")

func _0001_add_messages_contacts_up_db_sql() ([]byte, error) {
	return bindata_read(
		__0001_add_messages_contacts_up_db_sql,
		"0001_add_messages_contacts.up.db.sql",
	)
}

var __0002_add_state_down_db_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\x28\x2e\x49\x2c\x49\xb5\xe6\x02\x04\x00\x00\xff\xff\x93\x2f\x1f\x31\x12\x00\x00\x00")

func _0002_add_state_down_db_sql() ([]byte, error) {
	return bindata_read(
		__0002_add_state_down_db_sql,
		"0002_add_state.down.db.sql",
	)
}

var __0002_add_state_up_db_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x6c\xce\xb1\x0a\x83\x30\x10\xc6\xf1\x3d\x4f\xf1\x8d\x0a\xbe\x81\x93\x69\xaf\xe5\x20\x9c\xb4\x46\x70\xd3\xa2\xa1\x75\x51\x31\xf1\xfd\x0b\xb1\x42\x07\xb7\xe3\xf7\x0d\xf7\xbf\x3c\xa9\xb0\x04\x5b\x68\x43\xe0\x1b\xa4\xb4\xa0\x86\x2b\x5b\xc1\x87\x57\x70\x48\x14\x00\x8c\x03\xb4\x29\x75\x9c\xa5\x36\x26\x8b\xda\xbd\xd7\x79\x5b\xba\xb3\x69\x71\x6e\x3d\x73\xef\xa6\xa1\xed\xe7\x6d\x0a\xd0\x7c\x6f\x59\xec\x9f\xbb\x65\xee\x3f\x87\xab\x34\x57\xea\x97\x57\x0b\x3f\x6a\x02\xcb\x95\x9a\xbd\xcb\xa3\x94\xfd\x4a\xc6\x21\x3b\x4a\xb2\xf8\x37\xcd\xd5\x37\x00\x00\xff\xff\x5e\x55\xa1\x15\xd7\x00\x00\x00")

func _0002_add_state_up_db_sql() ([]byte, error) {
	return bindata_read(
		__0002_add_state_up_db_sql,
		"0002_add_state.up.db.sql",
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
	"0002_add_state.down.db.sql": _0002_add_state_down_db_sql,
	"0002_add_state.up.db.sql": _0002_add_state_up_db_sql,
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
	"0002_add_state.down.db.sql": &_bintree_t{_0002_add_state_down_db_sql, map[string]*_bintree_t{
	}},
	"0002_add_state.up.db.sql": &_bintree_t{_0002_add_state_up_db_sql, map[string]*_bintree_t{
	}},
}}
