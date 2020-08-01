package model

// 软删除 Reco 时从 Collection.RecoIDs 中删除相关 id, 但不清空 Reco.Collections,
// 从回收站恢复时询问用户是否重新添加到相关 Collection 中.

type Reco struct {
	ID          string   // primary key
	Collections []string // a file can be in different collections. []Collection.ID
	Message     string   // available only if Reco.Type is "NotFile"
	Links       []string // available only if Reco.Type is "NotFile"
	Tags        []string // []Tag.Name
	Name        string   `storm:"index"`
	Size        int64
	Type        string
	Checksum    string `storm:"unique"` // hex(sha256)
	Thumb       string // base64
	AccessCount int64
	AccessedAt  string `storm:"index"` // ISO8601
	CreatedAt   string `storm:"index"`
	UpdatedAt   string `storm:"index"`
	DeletedAt   string `storm:"index"`
}

type Tag struct {
	Name    string `storm:"id"`
	RecoIDs []string
}

type Collection struct {
	ID        string   // primary key
	Title     string   // `storm:"unique"` 要否限制唯一性？
	RecoIDs   []string // []Reco.ID
	CreatedAt string   `storm:"index"` // ISO8601
	UpdatedAt string   `storm:"index"`
}
