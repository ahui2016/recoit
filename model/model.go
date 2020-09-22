package model

import (
	"errors"

	"github.com/ahui2016/recoit/util"
)

// 软删除 Reco 时从 Collection.RecoIDs 中删除相关 id, 但不清空 Reco.Collections,
// 从回收站恢复时询问用户是否重新添加到相关 Collection 中.

// 当 Reco.FileType 的值为 NoFile 时，表示该 reco 不含文件。
const NoFile = "NoFile"

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
	if filename == "" {
		reco.FileType = NoFile
		return reco
	}
	reco.FileName = filename
	reco.FileType = util.TypeByFilename(filename)
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

type Collection struct {
	ID        string   // primary key
	Title     string   // `storm:"unique"` 要否限制唯一性？
	RecoIDs   []string // []Reco.ID // 允许用户排序(顶置)
	CreatedAt string   `storm:"index"` // ISO8601
	UpdatedAt string   `storm:"index"`
}
