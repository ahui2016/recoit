package model

import (
	"errors"
	"strings"

	"github.com/ahui2016/recoit/util"
)

// 软删除 Reco 时从 Box.RecoIDs 中删除相关 id, 但不清空 Reco.Boxes,
// 从回收站恢复时询问用户是否重新添加到相关 Box 中.

// MinLength of Reco.FileName if Reco.Type is File.
const MinLength = 3

// RecoType 是一个枚举类型，用来表示 Reco 的类型。
type RecoType string

// File .
const (
	Others RecoType = ""
	File   RecoType = "File"
	First  RecoType = "First"
)

// Reco .
type Reco struct {
	ID          string // primary key
	Type        RecoType
	Boxes       []string // a file can be in different boxes. []Box.ID
	Message     string
	Links       []string
	Tags        []string // []Tag.Name
	FileName    string   `storm:"index"`
	FileSize    int64
	Checksum    string `storm:"unique"` // hex(sha256)
	FileType    string
	AccessCount int64
	AccessedAt  string `storm:"index"` // ISO8601
	CreatedAt   string `storm:"index"`
	UpdatedAt   string `storm:"index"`
	DeletedAt   string `storm:"index"`
}

// NewReco .
func NewReco(recoType RecoType) *Reco {
	now := util.TimeNow()
	reco := &Reco{
		ID:        util.NewID(),
		CreatedAt: now,
		UpdatedAt: now,
	}
	reco.Type = recoType
	return reco
}

// NewFirstReco .
func NewFirstReco() *Reco {
	reco := NewReco(First)
	reco.ID = "1"
	return reco
}

// NewFile .
func NewFile(filename string) (*Reco, error) {
	reco := NewReco(File)
	reco.SetFileNameType(filename)
	return reco, nil
}

// SetFileNameType .
func (reco *Reco) SetFileNameType(filename string) error {
	filename = strings.TrimSpace(filename)
	if len(filename) < MinLength {
		return errors.New("filename is too short")
	}
	reco.FileName = filename
	reco.FileType = util.TypeByFilename(filename)
	return nil
}

// EqualContent .
func (reco *Reco) EqualContent(other *Reco) bool {
	if reco.ID != other.ID {
		return false
	}
	if util.SameSlice(reco.Boxes, other.Boxes) &&
		reco.Type == other.Type &&
		reco.Message == other.Message &&
		util.SameSlice(reco.Links, other.Links) &&
		util.SameSlice(reco.Tags, other.Tags) &&
		reco.FileName == other.FileName &&
		reco.FileSize == other.FileSize &&
		reco.FileType == other.FileType &&
		reco.Checksum == other.Checksum {
		return true
	}
	return false
}

type Tag struct {
	Name    string `storm:"id"`
	RecoIDs []string
}

func NewTag(name, id string) *Tag {
	return &Tag{
		name,
		[]string{id},
	}
}

func (tag *Tag) Add(id string) {
	if util.HasString(tag.RecoIDs, id) {
		return
	}
	tag.RecoIDs = append(tag.RecoIDs, id)
}

func (tag *Tag) Remove(id string) {
	i := util.StringIndex(tag.RecoIDs, id)
	if i < 0 {
		return
	}
	tag.RecoIDs = util.DeleteFromSlice(tag.RecoIDs, i)
}

// Box 像一个快递纸箱，可以装入各种不同类型的 Reco.
// 通常纸箱不会很大，但暂时不限制大小。
type Box struct {
	ID        string   // primary key
	Title     string   `storm:"unique"`
	RecoIDs   []string // []Reco.ID // 允许用户排序(顶置)
	CreatedAt string   `storm:"index"` // ISO8601
	UpdatedAt string   `storm:"index"`
}
