package main

import (
	"net/http"

	"github.com/ahui2016/recoit/model"
	"github.com/ahui2016/recoit/util"
	"github.com/asdine/storm/v3"
)

func accessUpdate(id string, count int64) error {
	reco := Reco{ID: id}
	if err := db.UpdateField(&reco, "AccessCount", count+1); err != nil {
		return err
	}
	return db.UpdateField(&reco, "AccessedAt", util.TimeNow())
}

func deleteReco(id string) error {
	reco := Reco{ID: id}
	return db.UpdateField(&reco, "DeletedAt", util.TimeNow())
}

func addTags(w http.ResponseWriter, tx storm.Node, tags []string, recoID string) error {
	for _, tagName := range tags {
		tag := new(Tag)
		err := tx.One("Name", tagName, tag)
		if err != nil && err != storm.ErrNotFound {
			return err
		}
		if err == storm.ErrNotFound {
			t := model.NewTag(tagName, recoID)
			if err := tx.Save(t); err != nil {
				return err
			}
			continue
		}
		// if found (err == nil)
		tag.Add(recoID)
		if err := tx.Update(tag); err != nil {
			return err
		}
	}
	return nil
}
