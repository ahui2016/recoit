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

func insertReco(w http.ResponseWriter, reco *Reco) error {
	tx, err := db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := tx.Save(reco); err != nil {
		return err
	}
	if err := addTags(w, tx, reco.Tags, reco.ID); err != nil {
		return err
	}
	return tx.Commit()
}

func deleteTags(w http.ResponseWriter, tx storm.Node, tagsToDelete []string, recoID string) error {
	for _, tagName := range tagsToDelete {
		tag := new(Tag)
		if err := tx.One("Name", tagName, tag); err != nil {
			return err
		}
		tag.Remove(recoID) // 每一个 tag 都与该 reco.ID 脱离关系
		return tx.Update(tag)
	}
	return nil
}

func addTags(w http.ResponseWriter, tx storm.Node, tags []string, recoID string) error {
	for _, tagName := range tags {
		tag := new(Tag)
		err := tx.One("Name", tagName, tag)
		if err != nil && err != storm.ErrNotFound {
			return err
		}
		if err == storm.ErrNotFound {
			aTag := model.NewTag(tagName, recoID)
			if err := tx.Save(aTag); err != nil {
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

func isFirstRecoExist() bool {
	var reco Reco
	err := db.One("ID", "1", &reco)
	if err != nil && err != storm.ErrNotFound {
		panic(err)
	}
	if err == storm.ErrNotFound {
		return false
	}
	return true
}

func getRecoByID(id string) (*Reco, error) {
	reco := new(Reco)
	err := db.One("ID", id, reco)
	return reco, err
}
