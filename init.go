package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/asdine/storm/v3"

	"github.com/ahui2016/recoit/database"
	"github.com/ahui2016/recoit/model"
	"github.com/ahui2016/recoit/session"
	"github.com/ahui2016/recoit/util"
)

const (
	databaseFolderName = "RecoitDB"
	databaseFileName   = "recoit.db"
	tempFolderName     = "RecoitTempDir"
	tempFileExt        = ".reco"
	cacheFolderName    = "RecoitCacheDir"
	staticFolder       = "static"
	maxAge             = 60 * 60 * 24 * 30 // 30 days
	secret             = "08-1303"
	passwordMaxTry     = 5
)

var (
	homeDir      string
	dbDefaultDir string
	dbPath       string
	tempDir      string
	cacheDir     string
	db           *storm.DB
)

var (
	passwordTry    = 0
	htmlFiles      = make(map[string]string)
	sessionManager = session.NewManager(maxAge)
)

type (
	Reco       = model.Reco
	Tag        = model.Tag
	Collection = model.Collection
)

func init() {
	homeDir = userHomeDir()
	dbDefaultDir = filepath.Join(homeDir, databaseFolderName)
	dbPath = filepath.Join(dbDefaultDir, databaseFileName)
	tempDir = filepath.Join(homeDir, tempFolderName)
	cacheDir = filepath.Join(homeDir, cacheFolderName)

	fillHtmlFiles()

	// open the db here, close the db in main().
	var err error
	db, err = database.Open(dbPath)
	if err != nil {
		panic(err)
	}
	log.Print(dbPath)
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

func jsonMessage(w http.ResponseWriter, message string, code int) {
	err := map[string]string{"message": message}
	jsonResponse(w, err, code)
}

// https://stackoverflow.com/questions/59763852/can-you-return-json-in-golang-http-error
func jsonResponse(w http.ResponseWriter, obj interface{}, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(obj)
}

func createFile(filePath string, src io.Reader) (int64, *os.File, error) {
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return 0, nil, err
	}
	size, err := io.Copy(f, src)
	if err != nil {
		return 0, nil, err
	}
	return size, f, nil
}
