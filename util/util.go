package util

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"math/big"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/ahui2016/recoit/graphics"
)

const (
	// 需要根据服务器的具体时区来设定正确的时区
	ISO8601 = "2006-01-02T15:04:05.999+08:00"
)

func NewID() string {
	var max int64 = 100_000_000
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		panic(err)
	}
	timestamp := time.Now().Unix()
	idInt64 := timestamp*max + n.Int64()
	return strconv.FormatInt(idInt64, 36)
}

func Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func Base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

func NewFileScanner(fullPath string) (*bufio.Scanner, *os.File, error) {
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, nil, err
	}
	return bufio.NewScanner(file), file, nil
}

func GetPathsByExt(dir, ext string) ([]string, error) {
	pattern := filepath.Join(dir, "*"+ext)
	filePaths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(filePaths)
	return filePaths, nil
}

func TimestampFilename(ext string) string {
	name := strconv.FormatInt(time.Now().UnixNano(), 10)
	return name + ext
}

func BufWriteln(w *bufio.Writer, box64 string) (err error) {
	_, err = w.WriteString(box64 + "\n")
	return
}

func DeleteFiles(filePaths []string) error {
	for _, f := range filePaths {
		if err := os.Remove(f); err != nil {
			return err
		}
	}
	return nil
}

func TimeNow() string {
	return time.Now().Format(ISO8601)
}

func Sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func HasString(slice []string, item string) bool {
	i := StringIndex(slice, item)
	if i < 0 {
		return false
	}
	return true
}

func StringIndex(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}

func DeleteFromSlice(slice []string, i int) []string {
	return append(slice[:i], slice[i+1:]...)
}

func PathIsNotExist(name string) bool {
	_, err := os.Lstat(name)
	if os.IsNotExist(err) {
		return true
	}
	if err != nil {
		panic(err)
	}
	return false
}

func PathIsExist(name string) bool {
	return !PathIsNotExist(name)
}

func CreateFile(filePath string, src io.Reader) (int64, *os.File, error) {
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return 0, nil, err
	}
	size, err := io.Copy(f, src)
	if err != nil {
		return 0, nil, err
	}
	return size, f, nil
}

func CreateThumb(imgPath, thumbPath string) error {
	buf, err := graphics.Thumbnail(imgPath)
	if err != nil {
		return err
	}
	if _, f, err := CreateFile(thumbPath, buf); err != nil {
		return err
	} else {
		f.Close()
	}
	return nil
}

func TypeByFilename(filename string) string {
	return mime.TypeByExtension(filepath.Ext(filename))
}

// DifferentSlice 对比新旧 slice 的差异，并返回需要新增的项目与需要删除的项目。
func DifferentSlice(oldSlice, newSlice []string) (toAdd, toDelete []string) {
	// newTags 里有，oldTags 里没有的，需要添加到数据库。
	for _, newItem := range newSlice {
		if !HasString(oldSlice, newItem) {
			toAdd = append(toAdd, newItem)
		}
	}
	// oldTags 里有，newTags 里没有的，需要从数据库中删除。
	for _, oldItem := range oldSlice {
		if !HasString(newSlice, oldItem) {
			toDelete = append(toDelete, oldItem)
		}
	}
	return
}

// SameSlice 判断两个 string slice 的内容是否相同，不限顺序。
func SameSlice(a, b []string) bool {
	newItems, missingItems := DifferentSlice(a, b)
	if newItems == nil && missingItems == nil {
		return true
	}
	return false
}

func RandBool() bool {
	max := big.NewInt(2)
	n, _ := rand.Int(rand.Reader, max)
	if n.Int64() == 0 {
		return false
	}
	return true
}

func RandomString() string {
	return Base64Encode(RandomBytes())
}

func RandomBytes() []byte {
	someBytes := make([]byte, 255)
	if _, err := rand.Read(someBytes); err != nil {
		panic(err) // 因为这里有错误的可能性极小, 因此偷懒不处理.
	}
	return someBytes
}
