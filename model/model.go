package model

import (
	"errors"
	"strings"

	"github.com/ahui2016/recoit/util"
)

// 软删除 Reco 时从 Collection.RecoIDs 中删除相关 id, 但不清空 Reco.Collections,
// 从回收站恢复时询问用户是否重新添加到相关 Collection 中.

// 当 Reco.FileType 的值为 NotFile 时，表示该 reco 不含文件。
const NotFile = "NotFile"

const MinLength = 3

type Reco struct {
	ID          string   // primary key
	Collections []string // a file can be in different collections. []Collection.ID
	Message     string
	Links       []string
	Tags        []string // []Tag.Name
	FileName    string   `storm:"index"`
	FileSize    int64
	FileType    string
	Checksum    string `storm:"unique"` // hex(sha256)
	AccessCount int64
	AccessedAt  string `storm:"index"` // ISO8601
	CreatedAt   string `storm:"index"`
	UpdatedAt   string `storm:"index"`
	DeletedAt   string `storm:"index"`
}

func NewReco(filename string) *Reco {
	now := util.TimeNow()
	reco := &Reco{
		ID:        util.NewID(),
		CreatedAt: now,
		UpdatedAt: now,
	}
	reco.FileName = strings.TrimSpace(filename)
	if reco.FileName == "" {
		reco.FileType = NotFile
		return reco
	}
	reco.FileType = util.TypeByFilename(filename)
	return reco
}

func NewFirstReco() *Reco {
	reco := NewReco("")
	reco.ID = "1"
	return reco
}

func NewFile(filename string) (*Reco, error) {
	if len(filename) < MinLength {
		return nil, errors.New("filename is too short")
	}
	return NewReco(filename), nil
}

func (reco *Reco) SetFileNameType(filename string) error {
	if len(filename) < MinLength {
		return errors.New("filename is too short")
	}
	reco.FileName = filename
	reco.FileType = util.TypeByFilename(filename)
	return nil
}

func (a *Reco) EqualContent(b *Reco) bool {
	if a.ID != b.ID {
		return false
	}
	if util.SameSlice(a.Collections, b.Collections) &&
		a.Message == b.Message &&
		util.SameSlice(a.Links, b.Links) &&
		util.SameSlice(a.Tags, b.Tags) &&
		a.FileName == b.FileName &&
		a.FileSize == b.FileSize &&
		a.FileType == b.FileType &&
		a.Checksum == b.Checksum {
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

type Collection struct {
	ID        string   // primary key
	Title     string   `storm:"unique"`
	RecoIDs   []string // []Reco.ID // 允许用户排序(顶置)
	CreatedAt string   `storm:"index"` // ISO8601
	UpdatedAt string   `storm:"index"`
}
