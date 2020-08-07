package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"

	"github.com/ahui2016/recoit/model"
	"github.com/ahui2016/recoit/util"
	"github.com/asdine/storm/v3"
)

func main() {
	defer db.Close()

	fs := http.FileServer(http.Dir("public/"))
	http.Handle("/public/", http.StripPrefix("/public/", fs))

	http.HandleFunc("/", homePage)
	// http.HandleFunc("/api/new-reco", newRecoHandler)
	http.HandleFunc("/api/upload-file", uploadHandler)
	http.HandleFunc("/api/checksum", checksumHandler)

	addr := "127.0.0.1:80"
	fmt.Println(addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func homePage(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		fallthrough
	case "/home":
		fmt.Fprint(w, htmlFiles["new-file"])
		// http.Redirect(w, r, "/search", 302)
	default:
		jsonMessage(w, "Not Found", 404)
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	var maxBytes int64 = 1024 * 1024 * 3 // 3 MB
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	file, fileHeader, err := r.FormFile("file")
	if checkErr(w, err, 500) {
		return
	}
	defer file.Close()

	// 将文件内容全部读入内存
	fileContents, err := ioutil.ReadAll(file)
	if checkErr(w, err, 500) {
		return
	}

	// 根据文件内容生成 checksum 并检查其是否正确
	if util.Sha256Hex(fileContents) != r.FormValue("checksum") {
		jsonMessage(w, "Error Checksum Unmatching", 400)
		return
	}

	// 新建一个 Reco, 获得其 ID
	reco := model.NewReco(r.FormValue("file-name"))

	reco.Checksum = r.FormValue("checksum")
	reco.FileSize = fileHeader.Size

	// 以 reco.ID 作为文件名生成临时文件
	filePath := filepath.Join(tempDir, reco.ID+tempFileExt)
	err = ioutil.WriteFile(filePath, fileContents, 0600)
	if checkErr(w, err, 500) {
		return
	}

	// 添加标签到 Reco, 后续还要添加 Reco.ID 到 Tag 数据表。
	fileTags := []byte(r.FormValue("file-tags"))
	err = json.Unmarshal(fileTags, &reco.Tags)
	if checkErr(w, err, 500) {
		return
	}

	// TODO: send to IBM COS

	// 开始一个事务
	tx, err := db.Begin(true)
	if checkErr(w, err, 500) {
		return
	}
	defer tx.Rollback()

	// Save the reco.
	if checkErr(w, tx.Save(reco), 500) {
		return
	}

	// Save tags.
	var tag Tag
	for _, tagName := range reco.Tags {
		err := tx.One("Name", tagName, &tag)
		if err != nil && err != storm.ErrNotFound {
			jsonResponse(w, err, 500)
			return
		}
		if err == storm.ErrNotFound {
			t := model.NewTag(tagName, reco.ID)
			if checkErr(w, tx.Save(t), 500) {
				return
			}
			continue
		}
		// if found (err == nil)
		tag.Add(reco.ID)
		if checkErr(w, tx.Update(tag), 500) {
			return
		}
	}
	tx.Commit()
}

func checksumHandler(w http.ResponseWriter, r *http.Request) {
	hashHex := r.FormValue("hashHex")
	var reco Reco
	err := db.One("Checksum", hashHex, &reco)
	if err == storm.ErrNotFound {
		jsonMessage(w, "OK", 200)
		return
	}
	if checkErr(w, err, 500) {
		return
	}

	// 正常找到已存在 hashHex, 表示发生文件冲突。
	jsonMessage(w, "Checksum Already Exists", 400)
}
