package database

import (
	"errors"
	"io/ioutil"
	"log"

	"github.com/ahui2016/recoit/aesgcm"
	"github.com/ahui2016/recoit/ibm"
	"github.com/ahui2016/recoit/model"
	"github.com/ahui2016/recoit/util"
	"github.com/asdine/storm/v3"
)

type (
	Reco       = model.Reco
	Tag        = model.Tag
	Collection = model.Collection
)

// DB 将数据库、加密、云储存三大功能汇于一身。
type DB struct {
	DB  *storm.DB
	GCM *aesgcm.AEAD
	COS *ibm.COS
}

func (db *DB) Open(dbPath string) (err error) {
	if db.DB, err = storm.Open(dbPath); err != nil {
		return err
	}
	log.Print(dbPath)
	return nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}

// InsertFirstReco 向数据库插入第一条数据，这个数据包含了该数据库的密码。
// 一旦操作成功，从此必须输入正确密码 (DB.Login) 才能读写数据库。
func (db *DB) InsertFirstReco(passphrase string) error {
	if db.IsFirstRecoExist() {
		return errors.New("已存在账号，不可重复创建")
	}
	if passphrase == "" {
		return errors.New("password is empty")
	}
	firstReco := newFirstReco(passphrase)
	db.createIndexes()
	return db.DB.Save(firstReco)
}

// 用 userKey 来加密 masterKey.
// userKey 用来加密解密 firstReco.Message,
// masterKey 用来加密解密其他数据。
func newFirstReco(passphrase string) *Reco {
	userKey := aesgcm.Sha256(passphrase)
	userGCM := aesgcm.NewGCM(userKey)
	masterKey := aesgcm.RandomKey()
	cipherMasterKey := userGCM.Encrypt(masterKey)

	firstReco := model.NewFirstReco()
	firstReco.Message = util.Base64Encode(cipherMasterKey)
	return firstReco
}

func (db *DB) createIndexes() error {
	if err := db.DB.Init(&Reco{}); err != nil {
		return err
	}
	if err := db.DB.Init(&Tag{}); err != nil {
		return err
	}
	if err := db.DB.Init(&Collection{}); err != nil {
		return err
	}
	return nil
}

func (db *DB) LoginLoadSettings(passphrase, settingsPath string) error {
	err := db.Login(passphrase)
	if err != nil {
		return err
	}
	return db.loadSettings(settingsPath)
}

func (db *DB) Login(passphrase string) error {
	if passphrase == "" {
		return errors.New("password is empty")
	}
	reco, err := db.GetRecoByID("1")
	if err != nil {
		return err
	}
	db.GCM, err = decryptFirstReco(passphrase, reco.Message)
	if err != nil {
		return err
	}
	return nil
}

func decryptFirstReco(passphrase, key64 string) (*aesgcm.AEAD, error) {
	userKey := aesgcm.Sha256(passphrase)
	userGCM := aesgcm.NewGCM(userKey)
	cipherMasterKey, err := util.Base64Decode(key64)
	if err != nil {
		return nil, err
	}
	masterKey, err := userGCM.Decrypt(cipherMasterKey)
	if err != nil {
		return nil, err
	}
	// 解密成功，获得 masterKey
	gcm := aesgcm.NewGCM(masterKey)
	return gcm, nil
}

func (db *DB) SetupCloud(settings *ibm.Settings, settingsPath string) error {
	if db.GCM == nil {
		return errors.New("require login")
	}
	cos := ibm.NewCOS(settings)

	// 检查 settings 是否正确。
	if err := cos.TryUploadDelete(); err != nil {
		return err
	}

	// 加密并写入硬盘。
	settingsJSON := settings.Encode()
	encrypted := db.GCM.Encrypt(settingsJSON)
	if err := ioutil.WriteFile(settingsPath, encrypted, 0600); err != nil {
		return err
	}

	// 云储存设置成功, 从此 db.COS != nil
	db.COS = cos
	return nil
}

// 检查云储存的 settings 是否已经保存在本地，如果是，则直接从本地导入 settings.
// 如果本地没有 settings, 则不进行任何操作。
func (db *DB) loadSettings(settingsPath string) error {
	if util.PathIsNotExist(settingsPath) {
		return nil
	}
	if db.GCM == nil {
		return errors.New("require login")
	}
	encryptedSettings, err := ioutil.ReadFile(settingsPath)
	if err != nil {
		return err
	}
	settingsJSON, err := db.GCM.Decrypt(encryptedSettings)
	if err != nil {
		return err
	}
	settings := ibm.NewSettingsFromJSON(settingsJSON)
	db.COS = ibm.NewCOS(settings)
	return nil
}

func (db *DB) IsFirstRecoExist() bool {
	var reco Reco
	err := db.DB.One("ID", "1", &reco)
	if err != nil && err != storm.ErrNotFound {
		panic(err)
	}
	if err == storm.ErrNotFound {
		return false
	}
	return true
}

func (db *DB) GetRecoByID(id string) (*Reco, error) {
	reco := new(Reco)
	err := db.DB.One("ID", id, reco)
	return reco, err
}

func (db *DB) AccessCountUp(id string, count int64) error {
	reco := Reco{ID: id}
	if err := db.DB.UpdateField(&reco, "AccessCount", count+1); err != nil {
		return err
	}
	return db.DB.UpdateField(&reco, "AccessedAt", util.TimeNow())
}

func (db *DB) DeleteReco(id string) error {
	reco := Reco{ID: id}
	return db.DB.UpdateField(&reco, "DeletedAt", util.TimeNow())
}

func (db *DB) InsertReco(reco *Reco) error {
	tx, err := db.DB.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := tx.Save(reco); err != nil {
		return err
	}
	if err := addTags(tx, reco.Tags, reco.ID); err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) UpdateReco(oldReco, reco *Reco) error {
	tx, err := db.DB.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 删除再重写，相当于一次完全的更新。
	// 因为 storm 的 update 方法不可更新空值。
	if err := tx.DeleteStruct(oldReco); err != nil {
		return err
	}
	if err := tx.Save(reco); err != nil {
		return err
	}

	toAdd, toDelete := util.DifferentSlice(oldReco.Tags, reco.Tags)

	// 删除标签（从 tag.RecoIDs 里删除 id）
	if err := deleteTags(tx, toDelete, reco.ID); err != nil {
		return err
	}

	// 添加标签（将 id 添加到 tag.RecoIDs 里）
	if err := addTags(tx, toAdd, reco.ID); err != nil {
		return err
	}

	tx.Commit()
	return nil
}

func addTags(tx storm.Node, tags []string, recoID string) error {
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

func deleteTags(tx storm.Node, tagsToDelete []string, recoID string) error {
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

func (db *DB) GetTagByName(name string) (*Tag, error) {
	tag := new(Tag)
	err := db.DB.One("Name", name, tag)
	return tag, err
}
