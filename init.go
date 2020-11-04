package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ahui2016/goutil"
	"github.com/ahui2016/recoit/database"
	"github.com/ahui2016/recoit/graphics"
	"github.com/ahui2016/recoit/model"
)

const (
	recoitDataFolderName = "recoit_data_folder"
	databaseFolderName   = "RecoitDB"         // inside "recoit_data_folder"
	databaseFileName     = "recoit.db"        // inside "RecoitDB"
	cosSettingsFileName  = "settings.cloud"   // inside "RecoitDB"
	cacheFolderName      = "RecoitCacheDir"   // inside "recoit_data_folder"
	cacheThumbFolderName = "RecoitCacheThumb" // inside "recoit_data_folder"
	tempFolderName       = "RecoitTempDir"    // inside "recoit_data_folder"
	recoFileExt          = ".reco"
	thumbFileExt         = ".small"
	staticFolder         = "static"
	passwordMaxTry       = 5

	// 3 MB, for http.MaxBytesReader
	maxBytes int64 = 1024 * 1024 * 3

	// maxAge = 60 * 60 * 24 * 30
	// 30 days, for session
	maxAge = 60 * 30

	// 500KB, 当一个图片小于 smallImageSize, 它就是小图片，小图片不需要生成缩略图。
	smallImageSize = 500 * 1024
)

var (
	recoitDataDir   string
	dbPath          string
	cosSettingsPath string
	tempDir         string
	cacheDir        string
	cacheThumbDir   string
)

var (
	passwordTry = 0
	HTML        = make(map[string]string)
	db          = new(database.DB)
)

// Types from model.
type (
	Reco = model.Reco
	Tag  = model.Tag
	Box  = model.Box
)

func init() {
	recoitDataDir = filepath.Join(userHomeDir(), recoitDataFolderName)
	dbDefaultDir := filepath.Join(recoitDataDir, databaseFolderName)
	dbPath = filepath.Join(dbDefaultDir, databaseFileName)
	cosSettingsPath = filepath.Join(dbDefaultDir, cosSettingsFileName)
	tempDir = filepath.Join(recoitDataDir, tempFolderName)
	cacheDir = filepath.Join(recoitDataDir, cacheFolderName)
	cacheThumbDir = filepath.Join(recoitDataDir, cacheThumbFolderName)

	fillHTML()
	goutil.MustMkdir(dbDefaultDir)
	goutil.MustMkdir(tempDir)
	goutil.MustMkdir(cacheDir)
	goutil.MustMkdir(cacheThumbDir)

	// open the db here, close the db in main().
	if err := db.Open(maxAge, dbPath, cosSettingsPath); err != nil {
		panic(err)
	}
}

func userHomeDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return homeDir
}

// fillHTML 把读取 html 文件的内容，塞进 HTML (map[string]string)。
// 目的是方便以字符串的形式把 html 文件直接喂给 http.ResponseWriter.
func fillHTML() {
	filePaths, err := goutil.GetFilesByExt(staticFolder, ".html")
	if err != nil {
		panic(err)
	}

	for _, path := range filePaths {
		base := filepath.Base(path)
		name := strings.TrimSuffix(base, ".html")
		html, err := ioutil.ReadFile(path)
		if err != nil {
			panic(err)
		}
		HTML[name] = string(html)
	}
}

func tempFilePath(id string) string {
	return filepath.Join(tempDir, addRecoExt(id))
}

func cacheFilePath(id string) string {
	return filepath.Join(cacheDir, addRecoExt(id))
}

func cacheThumbPath(id string) string {
	return filepath.Join(cacheThumbDir, id+thumbFileExt)
}

// writeCacheFile 在服务器保留缓存文件，如果是图片则顺便生成缩略图，
// 如果是图片并且不是 gif 动图，则压缩图片尺寸。
// 被压缩的图片与未经处理的文件保存在不同的文件夹。
func writeCacheFile(file *Reco, fileContents []byte) error {
	tempPath := tempFilePath(file.ID)
	cachePath := cacheFilePath(file.ID)
	thumbPath := cacheThumbPath(file.ID)

	// 当且只当是图片但不是 gif, 并且图片体积大于极限时，才压缩图片尺寸。
	if file.FileSize > smallImageSize && file.IsImage() && file.IsNotGIF() {
		buf, err := graphics.ResizeLimit(fileContents)
		if err != nil {
			return err
		}
		if err := goutil.CreateFile(cachePath, buf); err != nil {
			return err
		}
	} else {
		// 否则就直接写文件。
		err := ioutil.WriteFile(tempPath, fileContents, 0600)
		if err != nil {
			return err
		}
	}

	// 如果是图片则一律生成缩略图
	if file.IsImage() {
		err := goutil.BytesToThumb(fileContents, thumbPath)
		if err != nil {
			return err
		}
	}
	return nil
}

// addRecoExt adds '.reco' to name.
func addRecoExt(name string) string {
	return name + recoFileExt
}

// cacheFileUrl 返回前端访问缓存文件的 url (通常是被压缩尺寸的图片)
func cacheFileURL(name string) string {
	return "/cache/" + addRecoExt(name)
}

// tempFileUrl 返回前端访问临时文件的 url (通常是原图或原文件)
func tempFileURL(name string) string {
	return "/temp/" + addRecoExt(name)
}

// checkIDEmpty checks if the ID is empty or not.
func checkIDEmpty(w http.ResponseWriter, id string) bool {
	if id == "" {
		goutil.JsonMessage(w, "id is empty", 400)
		return true
	}
	return false
}
