package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ahui2016/recoit/graphics"
	"github.com/ahui2016/recoit/model"
	"github.com/ahui2016/recoit/util"
	"github.com/asdine/storm/v3"
)

func main() {
	defer db.Close()

	fs := http.FileServer(http.Dir("public/"))
	http.Handle("/public/", http.StripPrefix("/public/", fs))

	http.HandleFunc("/", homePage)
	http.HandleFunc("/api/new-reco", newRecoHandler)
	// http.HandleFunc("/api/upload", uploadHandler)
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
	var maxBytes int64 = 1024 * 1024 // 1 MB
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	// reader, err := r.MultipartReader()
	// if checkErr(w, err, 500) {
	// 	return
	// }
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
	jsonMessage(w, "ErrFound", 400)
}

func newRecoHandler(w http.ResponseWriter, r *http.Request) {
	var maxBytes int64 = 1024 * 1024 * 10 // 10 MB
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	reader, err := r.MultipartReader()
	if checkErr(w, err, 500) {
		return
	}

	tx, err := db.Begin(true)
	if checkErr(w, err, 500) {
		return
	}
	defer tx.Rollback()

	reco := model.NewReco()

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if checkErr(w, err, 500) {
			return
		}

		contents, err := ioutil.ReadAll(part)
		if checkErr(w, err, 500) {
			return
		}

		log.Print(part.FormName())

		switch formName := part.FormName(); formName {
		case "user":
			reco.User = string(contents)
		case "message":
			reco.User = string(contents)
		case "tags":
			err := json.Unmarshal(contents, &reco.Tags)
			if checkErr(w, err, 500) {
				return
			}
		case "links":
			err := json.Unmarshal(contents, &reco.Links)
			if checkErr(w, err, 500) {
				return
			}
		case "file":
			log.Print(part.FileName())
			file := model.NewFile(part.FileName(), reco.ID)
			file.Size = int64(len(contents))
			file.Checksum = util.Sha256Hex(contents)

			filePath := filepath.Join(tempDir, file.ID+tempFileExt)
			err := ioutil.WriteFile(filePath, contents, 0600)
			if checkErr(w, err, 500) {
				return
			}

			if strings.HasPrefix(file.Type, "image/") {
				thumb, err := graphics.Thumbnail(filePath)
				if checkErr(w, err, 500) {
					return
				}
				file.Thumb = thumb
			}

			// TODO: send to IBM COS

			if checkErr(w, tx.Save(file), 500) {
				return
			}
			reco.Files = append(reco.Files, file.ID)
		}
	}

	if checkErr(w, tx.Save(reco), 500) {
		return
	}
	if checkErr(w, tx.Commit(), 500) {
		return
	}

	jsonResponse(w, reco, 200)
}
