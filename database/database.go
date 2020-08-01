package database

import "github.com/asdine/storm/v3"

func Open(path string) (*storm.DB, error) {
	return storm.Open(path)
}
