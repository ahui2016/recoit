/*
Package cloud provides Cloud Object Storage interface.
*/
package cloud

import (
	"encoding/json"
	"io"
)

// Provider 是一个枚举类型，用来表示 COS 的服务商。
type Provider string

// IBM .
const (
	IBM Provider = "IBM"
)

// ObjectStorage .
type ObjectStorage interface {
	PutObject(string, io.ReadSeeker) error
	GetObjectBody(string) (io.ReadCloser, error)
	DeleteObject(string) error
	TryUploadDelete() error
}

// Settings .
type Settings interface {
	GetProvider() Provider
	Encode() []byte // encode to JSON
	NewCOS() ObjectStorage
}

// Which 用来帮助从 settings 的 json 中找出 provider.
type Which struct {
	Provider Provider
}

// GetProvider .
func GetProviderFromJSON(settingsJSON []byte) Provider {
	which := new(Which)
	if err := json.Unmarshal(settingsJSON, which); err != nil {
		panic(err)
	}
	return which.Provider
}
