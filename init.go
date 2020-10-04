package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ahui2016/recoit/database"
	"github.com/ahui2016/recoit/model"
	"github.com/ahui2016/recoit/session"
	"github.com/ahui2016/recoit/util"
)

const (
	recoitDataFolderName   = "recoit_data_folder"
	databaseFolderName     = "RecoitDB"         // inside "recoit_data_folder"
	databaseFileName       = "recoit.db"        // inside "RecoitDB"
	ibmCosSettingsFileName = "settings.ibm"     // inside "RecoitDB"
	cacheFolderName        = "RecoitCacheDir"   // inside "recoit_data_folder"
	cacheThumbFolderName   = "RecoitCacheThumb" // inside "recoit_data_folder"
	tempFolderName         = "RecoitTempDir"    // inside "recoit_data_folder"
	tempFileExt            = ".reco"
	thumbFileExt           = ".small"
	staticFolder           = "static"
	maxAge                 = 60 * 60 * 24 * 30 // 30 days
	secret                 = "08-1303"
	passwordMaxTry         = 5
)

var (
	recoitDataDir   string
	dbPath          string
	ibmSettingsPath string
	tempDir         string
	cacheDir        string
	cacheThumbDir   string
)

var (
	passwordTry    = 0
	htmlFiles      = make(map[string]string)
	sessionManager = session.NewManager(maxAge)
	db             = new(database.DB)
)

type (
	Reco       = model.Reco
	Tag        = model.Tag
	Collection = model.Collection
)

func init() {
	recoitDataDir = filepath.Join(userHomeDir(), recoitDataFolderName)
	dbDefaultDir := filepath.Join(recoitDataDir, databaseFolderName)
	dbPath = filepath.Join(dbDefaultDir, databaseFileName)
	ibmSettingsPath = filepath.Join(dbDefaultDir, ibmCosSettingsFileName)
	tempDir = filepath.Join(recoitDataDir, tempFolderName)
	cacheDir = filepath.Join(recoitDataDir, cacheFolderName)
	cacheThumbDir = filepath.Join(recoitDataDir, cacheThumbFolderName)

	fillHtmlFiles()
	mustMkdir(dbDefaultDir)
	mustMkdir(tempDir)
	mustMkdir(cacheDir)
	mustMkdir(cacheThumbDir)

	// open the db here, close the db in main().
	if err := db.Open(dbPath); err != nil {
		panic(err)
	}
}

func isCosExist(w http.ResponseWriter) bool {
	if cos == nil {
		jsonMessage(w, "cos is null", 400)
		return false
	}
	return true
}

func mustMkdir(name string) {
	if util.PathIsNotExist(name) {
		if err := os.Mkdir(name, 0600); err != nil {
			panic(err)
		}
	}
}

func userHomeDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return homeDir
}

func fillHtmlFiles() {
	filePaths, err := util.GetPathsByExt(staticFolder, ".html")
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
		htmlFiles[name] = string(html)
	}
}

func checkErr(w http.ResponseWriter, err error, code int) bool {
	if err != nil {
		log.Println(err)
		jsonMessage(w, err.Error(), code)
		return true
	}
	return false
}

func jsonMsgOK(w http.ResponseWriter) {
	jsonMessage(w, "OK", 200)
}

func jsonMsg404(w http.ResponseWriter) {
	jsonMessage(w, "Not Found", 404)
}

func jsonMessage(w http.ResponseWriter, message string, code int) {
	msg := map[string]string{"message": message}
	jsonResponse(w, msg, code)
}

// https://stackoverflow.com/questions/59763852/can-you-return-json-in-golang-http-error
func jsonResponse(w http.ResponseWriter, obj interface{}, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(obj)
}

func cacheFilePath(id string) string {
	return filepath.Join(cacheDir, id+tempFileExt)
}

func cacheThumbPath(id string) string {
	return filepath.Join(cacheThumbDir, id+thumbFileExt)
}

func writeCacheFile(file *Reco, fileContents []byte) error {
	filePath := cacheFilePath(file.ID)
	if err := ioutil.WriteFile(filePath, fileContents, 0600); err != nil {
		return err
	}
	// 如果该文件是图片，则顺便生成缩略图。
	if strings.HasPrefix(file.FileType, "image") {
		thumbPath := cacheThumbPath(file.ID)
		if err := util.CreateThumb(filePath, thumbPath); err != nil {
			return err
		}
	}
	return nil
}
