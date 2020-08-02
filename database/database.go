package database

import "github.com/asdine/storm/v3"

func Open(path string) (*storm.DB, error) {
	return storm.Open(path)
}

// InsertTagsReco inserts a new reco to the Reco table,
// and inserts the reco.ID to the Tag table.
// func InsertTagsReco

// func addNewTags(tx storm.Node, id string, tags []string) {}
