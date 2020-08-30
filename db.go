package main

import "github.com/ahui2016/recoit/util"

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
