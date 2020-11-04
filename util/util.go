package util

import (
	"bufio"
	"crypto/rand"
	"io/ioutil"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/ahui2016/goutil"
)

// TimestampFilename .
func TimestampFilename(ext string) string {
	name := strconv.FormatInt(time.Now().UnixNano(), 10)
	return name + ext
}

// BufWriteln .
func BufWriteln(w *bufio.Writer, box64 string) (err error) {
	_, err = w.WriteString(box64 + "\n")
	return
}

// DeleteFiles .
func DeleteFiles(filePaths []string) error {
	for _, f := range filePaths {
		if err := os.Remove(f); err != nil {
			return err
		}
	}
	return nil
}

// HasString .
func HasString(slice []string, item string) bool {
	i := StringIndex(slice, item)
	if i < 0 {
		return false
	}
	return true
}

// StringIndex .
func StringIndex(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}

// DeleteFromSlice .
func DeleteFromSlice(slice []string, i int) []string {
	return append(slice[:i], slice[i+1:]...)
}

// CreateThumb .
func CreateThumb(imgPath, thumbPath string) error {
	img, err := ioutil.ReadFile(imgPath)
	if err != nil {
		return err
	}
	return goutil.BytesToThumb(img, thumbPath)
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

// RandBool .
func RandBool() bool {
	max := big.NewInt(2)
	n, _ := rand.Int(rand.Reader, max)
	if n.Int64() == 0 {
		return false
	}
	return true
}

// RandomString .
func RandomString() string {
	return goutil.Base64Encode(RandomBytes())
}

// RandomBytes .
func RandomBytes() []byte {
	someBytes := make([]byte, 255)
	if _, err := rand.Read(someBytes); err != nil {
		panic(err) // 因为这里有错误的可能性极小, 因此偷懒不处理.
	}
	return someBytes
}
