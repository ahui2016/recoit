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

// 各种 RecoType.
const (
	Others RecoType = ""
	File   RecoType = "File"
	First  RecoType = "First"
)

// Reco .
type Reco struct {
	ID          string // primary key
	Type        RecoType
	Box         string // Box.ID
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
	return &Reco{
		ID:        util.NewID(),
		Type:      recoType,
		CreatedAt: now,
		UpdatedAt: now,
	}
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

// IsImage .
func (reco *Reco) IsImage() bool {
	return strings.HasPrefix(reco.FileType, "image")
}

// IsNotGIF .
func (reco *Reco) IsNotGIF() bool {
	return reco.FileType != "image/gif"
}

// EqualContent 判断两个 reco 的内容是否大致相同，要注意，
// 这里不判断 Box 和 Tags，因为这两项通常单独更新。
func (reco *Reco) EqualContent(other *Reco) bool {
	if reco.ID != other.ID {
		return false
	}
	if util.SameSlice(reco.Links, other.Links) &&
		reco.Type == other.Type &&
		reco.Message == other.Message &&
		reco.FileName == other.FileName &&
		reco.FileSize == other.FileSize &&
		reco.FileType == other.FileType &&
		reco.Checksum == other.Checksum {
		return true
	}
	return false
}

// Tag .
type Tag struct {
	Name    string `storm:"id"`
	RecoIDs []string
}

// NewTag .
func NewTag(name, id string) *Tag {
	return &Tag{
		name,
		[]string{id},
	}
}

// Add .
func (tag *Tag) Add(id string) {
	if util.HasString(tag.RecoIDs, id) {
		return
	}
	tag.RecoIDs = append(tag.RecoIDs, id)
}

// Remove .
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

// NewBox .
func NewBox(title string) *Box {
	now := util.TimeNow()
	return &Box{
		ID:        util.NewID(),
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Add 添加 reco-id, 如果实际上没有添加则返回 false.
func (box *Box) Add(recoID string) bool {
	if util.HasString(box.RecoIDs, recoID) {
		return false
	}
	box.RecoIDs = append(box.RecoIDs, recoID)
	box.UpdatedAt = util.TimeNow()
	return true
}

// Remove 删除 reco-id, 如果实际上没有删除则返回 false.
// 删除时不更新日期。
func (box *Box) Remove(recoID string) bool {
	i := util.StringIndex(box.RecoIDs, recoID)
	if i < 0 {
		return false
	}
	box.RecoIDs = util.DeleteFromSlice(box.RecoIDs, i)
	return true
}

// Rename updates the title of the box.
func (box *Box) Rename(title string) error {
	if box.Title == title {
		return errors.New("没有变化")
	}
	box.Title = title
	box.UpdatedAt = util.TimeNow()
	return nil
}
