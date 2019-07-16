// Code generated by go-bindata. DO NOT EDIT.
// sources:
// 000001_init.down.db.sql (18B)
// 000001_init.up.db.sql (586B)
// doc.go (377B)

package sqlite

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes  []byte
	info   os.FileInfo
	digest [sha256.Size]byte
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

func (fi bindataFileInfo) Name() string {
	return fi.name
}
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi bindataFileInfo) IsDir() bool {
	return false
}
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var __000001_initDownDbSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\x48\xce\x48\x2c\x29\xb6\xe6\x02\x04\x00\x00\xff\xff\x7f\xae\x7e\x3a\x12\x00\x00\x00")

func _000001_initDownDbSqlBytes() ([]byte, error) {
	return bindataRead(
		__000001_initDownDbSql,
		"000001_init.down.db.sql",
	)
}

func _000001_initDownDbSql() (*asset, error) {
	bytes, err := _000001_initDownDbSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "000001_init.down.db.sql", size: 18, mode: os.FileMode(0644), modTime: time.Unix(1563032486, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0x64, 0xe9, 0x26, 0xaf, 0xbe, 0x72, 0x9b, 0x33, 0x37, 0x23, 0x21, 0x46, 0xe7, 0xb1, 0xda, 0x60, 0xaa, 0xa6, 0x44, 0x29, 0xdd, 0xbc, 0x9b, 0x8d, 0x7c, 0x80, 0x9d, 0xf0, 0xb0, 0x3, 0x3f, 0xab}}
	return a, nil
}

var __000001_initUpDbSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x94\x91\x4f\x4b\x33\x31\x10\xc6\xef\xfd\x14\x03\xef\xa1\xaf\xe0\x41\x0f\xa2\xe0\x29\xdb\xa6\x34\x98\x6e\x4a\x3a\x6b\xed\x29\xc4\x64\xd0\xa5\xd9\x3f\xb8\xd9\x4a\xbf\xbd\xb8\x5b\x2b\x75\x51\xf4\xfc\xfc\x9e\x5f\x32\x33\x13\xcd\x19\x72\x40\x96\x48\x0e\x62\x06\xa9\x42\xe0\x0f\x62\x85\x2b\x70\xcf\x36\x36\xf0\x7f\x04\x00\x90\x7b\xb8\x67\x7a\x32\x67\x1a\x96\x5a\x2c\x98\xde\xc0\x1d\xdf\x74\x74\x9a\x49\x79\xde\x41\xa5\x2d\xe8\x88\x9d\x46\xae\x0a\xd5\xcb\x20\x83\x29\x9f\xb1\x4c\x22\x8c\xff\xd9\xcb\x9b\x6b\x7f\x35\xee\xe9\xb8\xaf\x09\x44\x8a\x5f\x24\xd6\xc5\x7c\x47\x90\x28\x25\x39\x4b\x87\x16\xd4\x19\x3f\x08\xf2\x82\x9a\x68\x8b\x1a\xa6\x0c\x39\x8a\x05\x1f\xd2\x93\x4c\x6b\x9e\xa2\x79\x4f\x57\xc8\x16\xcb\xbe\xda\xd6\xde\x46\xf2\xc6\xc6\xbf\x77\x3d\x05\xea\xbb\xc6\x85\xca\x6d\xcd\xce\x86\xf6\x74\x92\xa3\xe2\xa2\xaf\xd4\xed\x63\xc8\x9d\xd9\xd2\x1e\x12\xa9\x92\xc3\x1f\xca\x5d\x4e\xaf\xe4\x4d\x41\x4d\x63\x9f\xc8\xb8\xaa\x2d\xe3\x8f\x9e\x60\x9b\xdf\x3f\xda\xc1\x9f\xee\x32\x52\x19\x4d\xb7\xf4\xc3\x85\xbe\xc7\x3e\x88\xd1\x19\xac\x05\xce\x55\x86\xa0\xd5\x5a\x4c\x6f\x47\x6f\x01\x00\x00\xff\xff\x15\xc6\xf2\x45\x4a\x02\x00\x00")

func _000001_initUpDbSqlBytes() ([]byte, error) {
	return bindataRead(
		__000001_initUpDbSql,
		"000001_init.up.db.sql",
	)
}

func _000001_initUpDbSql() (*asset, error) {
	bytes, err := _000001_initUpDbSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "000001_init.up.db.sql", size: 586, mode: os.FileMode(0644), modTime: time.Unix(1563032468, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0x1d, 0x65, 0xba, 0x54, 0x51, 0x83, 0xc3, 0x7d, 0x18, 0x14, 0x95, 0x42, 0x9b, 0xd8, 0xbc, 0x77, 0xac, 0x15, 0x13, 0x33, 0x58, 0xa7, 0x5, 0x22, 0x93, 0x6e, 0xb4, 0xb8, 0x7c, 0xed, 0xeb, 0x74}}
	return a, nil
}

var _docGo = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x84\x8f\x3b\x72\xc3\x30\x0c\x44\x7b\x9d\x62\xc7\x8d\x9b\x88\x6c\x52\xa5\x4b\x99\x3e\x17\x80\x49\x88\xc4\x98\x1f\x85\x80\xfc\xb9\x7d\x46\x4e\x66\xe2\x2e\xed\x0e\xde\xe2\xad\xf7\xf8\xcc\xa2\x58\xa4\x30\x44\xd1\x38\xb0\x2a\x8d\x3b\x4e\x1c\x68\x53\xc6\x21\x89\xe5\xed\xe4\x42\xaf\x5e\x8d\x6c\xd3\x59\xaa\xaf\x92\x06\x19\xfb\xcb\xeb\x61\xf2\x1e\x81\xda\xd1\x90\xa9\xc5\xc2\x8f\x2e\x85\x1a\x0d\x93\x96\x70\x15\xcb\x20\xac\x83\x17\xb9\x39\xbc\x1b\x0a\x93\x1a\x2c\x93\x1d\x15\x96\x19\x81\x94\xf7\x9a\xa5\x0f\xa4\x3e\x9f\xa4\x45\x32\x72\x7b\xf4\xb1\x3c\x25\xbb\x61\xa0\x52\x38\x62\x19\xbd\x3e\x58\xa5\xca\x88\x32\x38\x58\x1f\xf7\x17\x90\x2a\x1b\x1a\x55\xd6\x9d\xcf\x74\x61\xb4\xfe\xfb\x1e\xd4\xe2\xff\x8b\x70\xed\xe3\xac\x20\x05\xdf\x56\x0e\xc6\xd1\x4d\xd3\x4a\xe1\x4c\x89\xf1\x73\x27\xbd\xe9\x34\x79\x9f\xfa\x5b\xe2\xc6\x3b\xf9\xec\x39\xaf\xe7\x04\xfd\x2a\x62\x8c\xb9\xc3\x39\xff\x87\xb9\xd4\xe1\xa6\xef\x00\x00\x00\xff\xff\xcd\x86\x58\x5c\x79\x01\x00\x00")

func docGoBytes() ([]byte, error) {
	return bindataRead(
		_docGo,
		"doc.go",
	)
}

func docGo() (*asset, error) {
	bytes, err := docGoBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "doc.go", size: 377, mode: os.FileMode(0644), modTime: time.Unix(1563032267, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0x94, 0xd1, 0x5c, 0x73, 0x47, 0x65, 0xf9, 0x6e, 0xa0, 0xee, 0xb, 0x3d, 0xbe, 0xff, 0xef, 0xae, 0xc9, 0x46, 0x21, 0x85, 0x12, 0x46, 0xa1, 0x73, 0x74, 0xca, 0x71, 0xb1, 0xe1, 0x69, 0xe1, 0x82}}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[canonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// AssetString returns the asset contents as a string (instead of a []byte).
func AssetString(name string) (string, error) {
	data, err := Asset(name)
	return string(data), err
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// MustAssetString is like AssetString but panics when Asset would return an
// error. It simplifies safe initialization of global variables.
func MustAssetString(name string) string {
	return string(MustAsset(name))
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[canonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetDigest returns the digest of the file with the given name. It returns an
// error if the asset could not be found or the digest could not be loaded.
func AssetDigest(name string) ([sha256.Size]byte, error) {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[canonicalName]; ok {
		a, err := f()
		if err != nil {
			return [sha256.Size]byte{}, fmt.Errorf("AssetDigest %s can't read by error: %v", name, err)
		}
		return a.digest, nil
	}
	return [sha256.Size]byte{}, fmt.Errorf("AssetDigest %s not found", name)
}

// Digests returns a map of all known files and their checksums.
func Digests() (map[string][sha256.Size]byte, error) {
	mp := make(map[string][sha256.Size]byte, len(_bindata))
	for name := range _bindata {
		a, err := _bindata[name]()
		if err != nil {
			return nil, err
		}
		mp[name] = a.digest
	}
	return mp, nil
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
var _bindata = map[string]func() (*asset, error){
	"000001_init.down.db.sql": _000001_initDownDbSql,

	"000001_init.up.db.sql": _000001_initUpDbSql,

	"doc.go": docGo,
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
// then AssetDir("data") would return []string{"foo.txt", "img"},
// AssetDir("data/img") would return []string{"a.png", "b.png"},
// AssetDir("foo.txt") and AssetDir("notexist") would return an error, and
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		canonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(canonicalName, "/")
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
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"000001_init.down.db.sql": &bintree{_000001_initDownDbSql, map[string]*bintree{}},
	"000001_init.up.db.sql":   &bintree{_000001_initUpDbSql, map[string]*bintree{}},
	"doc.go":                  &bintree{docGo, map[string]*bintree{}},
}}

// RestoreAsset restores an asset under the given directory.
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	return os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
}

// RestoreAssets restores an asset under the given directory recursively.
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(canonicalName, "/")...)...)
}
