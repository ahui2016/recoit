package main

import "testing"

func TestFindAll(t *testing.T) {
	defer db.Close()
	var all []Reco
	if err := db.DB.All(&all); err != nil {
		t.Fatal(err)
	}
	t.Log("all: ", all)

	var tags []Tag
	if err := db.DB.All(&tags); err != nil {
		t.Fatal(err)
	}
	t.Log("tags: ", tags)
}
