package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/ahui2016/recoit/model"
	"github.com/ahui2016/recoit/util"
	"github.com/asdine/storm/v3"
	"github.com/asdine/storm/v3/q"
)

func main() {
	defer db.Close()

	fs := http.FileServer(http.Dir("public"))
	http.Handle("/public/", http.StripPrefix("/public/", fs))

	cacheFS := http.FileServer(http.Dir(cacheDir))
	http.Handle("/cache/", http.StripPrefix("/cache/", cacheFS))

	thumbFS := http.FileServer(http.Dir(cacheThumbDir))
	http.Handle("/thumb/", http.StripPrefix("/thumb/", thumbFS))

	http.HandleFunc("/", homePage)
	http.HandleFunc("/index", indexPage)
	http.HandleFunc("/api/all-recos", getAllRecos)
	http.HandleFunc("/tag", tagPage)
	http.HandleFunc("/api/tag", getRecosByTag)
	http.HandleFunc("/add-file", addFilePage)
	http.HandleFunc("/edit-file", editFilePage)
	http.HandleFunc("/api/upload-file", uploadHandler)
	http.HandleFunc("/api/update-file", updateHandler)
	http.HandleFunc("/api/checksum", checksumHandler)
	http.HandleFunc("/api/reco", getRecoHandler)
	http.HandleFunc("/api/delete-reco", deleteRecoHandler)
	http.HandleFunc("/api/thumb", createThumbHandler)

	addr := "127.0.0.1:80"
	fmt.Println(addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func homePage(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		fallthrough
	case "/home":
		// fmt.Fprint(w, htmlFiles["index"])
		http.Redirect(w, r, "/index", 302)
	default:
		jsonMsg404(w)
	}
}

func indexPage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, htmlFiles["index"])
}

func addFilePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, htmlFiles["new-file"])
}

func editFilePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, htmlFiles["edit-file"])
}

func tagPage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, htmlFiles["tag"])
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
	filename := strings.TrimSpace(r.FormValue("file-name"))
	reco, err := model.NewFile(filename)
	if checkErr(w, err, 400) {
		return
	}

	reco.Checksum = r.FormValue("checksum")
	reco.FileSize = fileHeader.Size

	// 添加标签到 Reco, 后续还要添加 Reco.ID 到 Tag 数据表。
	fileTags := []byte(r.FormValue("file-tags"))
	if checkErr(w, json.Unmarshal(fileTags, &reco.Tags), 500) {
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
	if checkErr(w, addTags(w, tx, reco.Tags, reco.ID), 500) {
		return
	}

	tx.Commit()

	// 数据库操作成功，生成缓存文件（如果是图片，则顺便生成缩略图）。
	// 不可在数据库操作结束之前生成缓存文件，
	// 因为数据库操作发生错误时不应生成缓存文件。
	checkErr(w, writeCacheFile(reco, fileContents), 500)
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	var maxBytes int64 = 1024 * 1024 * 3 // 3 MB
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	// 获取数据库中的 reco
	id := r.FormValue("id")
	var reco Reco
	if checkErr(w, db.One("ID", id, &reco), 500) {
		return
	}

	// 为节省流量、减少出错，禁止上传相同文件。
	checksum := r.FormValue("checksum")
	if checksum != "" && reco.Checksum == checksum {
		jsonMessage(w, "Can not upload the same file", 400)
		return
	}

	// 本来还应该检查 checksum 的唯一性，但替换文件是低频操作，因此可以偷懒。

	file, fileHeader, err := r.FormFile("file")
	if err != nil && err != http.ErrMissingFile {
		jsonResponse(w, err, 500)
		return
	}
	if file != nil {
		defer file.Close()
	}

	var fileContents []byte
	// 当发生错误 http.ErrMissingFile 时，file 等于 null。
	if file != nil {
		// 将文件内容全部读入内存
		fileContents, err = ioutil.ReadAll(file)
		if checkErr(w, err, 500) {
			return
		}
		// 根据文件内容生成 checksum 并检查其是否正确
		if util.Sha256Hex(fileContents) != checksum {
			jsonMessage(w, "Error Checksum Unmatching", 400)
			return
		}
		reco.Checksum = checksum
		reco.FileSize = fileHeader.Size
	}

	if checkErr(w, reco.SetFileNameType(r.FormValue("file-name")), 400) {
		return
	}
	reco.Message = strings.TrimSpace(r.FormValue("description"))

	// TODO: to update reco.Collections

	fileLinks := []byte(r.FormValue("file-links"))
	if checkErr(w, json.Unmarshal(fileLinks, &reco.Links), 500) {
		return
	}

	fileTags := []byte(r.FormValue("file-tags"))
	if checkErr(w, json.Unmarshal(fileTags, &reco.Tags), 500) {
		return
	}

	// 至此，reco 已被更新，重新从数据库获取 reco 用来对比有无更新。
	var oldReco Reco
	if checkErr(w, db.One("ID", id, &oldReco), 500) {
		return
	}

	if oldReco.EqualContent(&reco) {
		jsonMessage(w, "无任何变化", 401)
		return
	}

	// 从这里开始，可以认为 reco 的内容与 oldReco 不同。
	reco.AccessCount++
	reco.AccessedAt = util.TimeNow()
	reco.UpdatedAt = reco.AccessedAt

	// TODO: update to IBM COS

	tx, err := db.Begin(true)
	if checkErr(w, err, 500) {
		return
	}
	defer tx.Rollback()

	// 删除再重写，相当于一次完全的更新。
	// 因为 storm 的 update 方法不可更新空值。
	if checkErr(w, tx.DeleteStruct(&oldReco), 500) {
		return
	}
	if checkErr(w, tx.Save(&reco), 500) {
		return
	}

	toAdd, toDelete := util.DifferentSlice(oldReco.Tags, reco.Tags)

	// 删除标签（从 tag.RecoIDs 里删除 id）
	for _, tagName := range toDelete {
		tag := new(Tag)
		if checkErr(w, tx.One("Name", tagName, tag), 500) {
			return
		}
		tag.Remove(reco.ID) // 每一个 tag 都与该 reco.ID 脱离关系
		if checkErr(w, tx.Update(tag), 500) {
			return
		}
	}

	// 添加标签（将 id 添加到 tag.RecoIDs 里）
	if checkErr(w, addTags(w, tx, toAdd, reco.ID), 500) {
		return
	}

	tx.Commit()

	// 更新缓存文件
	if reco.FileType != model.NoFile && reco.Checksum != oldReco.Checksum {
		if checkErr(w, writeCacheFile(&reco, fileContents), 500) {
			return
		}
	}
}

func checksumHandler(w http.ResponseWriter, r *http.Request) {
	hashHex := r.FormValue("hashHex")
	var reco Reco
	err := db.One("Checksum", hashHex, &reco)
	if err == storm.ErrNotFound {
		jsonMsgOK(w)
		return
	}
	if checkErr(w, err, 500) {
		return
	}

	// 正常找到已存在 hashHex, 表示发生文件冲突。
	jsonMessage(w, "Checksum Already Exists", 400)
}

func getRecoHandler(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	var reco Reco
	if checkErr(w, db.One("ID", id, &reco), 500) {
		return
	}
	if checkErr(w, accessUpdate(id, reco.AccessCount), 500) {
		return
	}
	reco.Checksum = ""
	jsonResponse(w, reco, 200)
}

func getAllRecos(w http.ResponseWriter, r *http.Request) {
	var all []Reco
	err := db.Select(q.Eq("DeletedAt", "")).OrderBy("UpdatedAt").Find(&all)
	if checkErr(w, err, 500) {
		return
	}
	for _, reco := range all {
		reco.Checksum = ""
	}
	jsonResponse(w, all, 200)
}

func deleteRecoHandler(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if checkErr(w, deleteReco(id), 500) {
		return
	}
	jsonMsgOK(w)
}

func createThumbHandler(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		jsonMessage(w, "id is empty", 400)
		return
	}

	imgPath := cacheFilePath(id)
	thumbPath := cacheThumbPath(id)

	// 验证这个文件是图片（省略，因为省略也不会出大问题）

	if util.PathIsNotExist(thumbPath) {

		// 如果 imgPath 不存在，则从 COS 获取文件（暂时省略，需要补充）

		err := util.CreateThumb(imgPath, thumbPath)
		if checkErr(w, err, 500) {
			return
		}
	}
	jsonMsgOK(w)
}

func getRecosByTag(w http.ResponseWriter, r *http.Request) {
	tagName := r.FormValue("tag")
	if tagName == "" {
		jsonMessage(w, "tag is empty", 400)
		return
	}
	tag := new(Tag)
	if checkErr(w, db.One("Name", tagName, tag), 500) {
		return
	}
	var recos []*Reco
	for _, id := range tag.RecoIDs {
		reco := new(Reco)
		if checkErr(w, db.One("ID", id, reco), 500) {
			return
		}
		reco.Checksum = ""
		recos = append(recos, reco)
	}
	jsonResponse(w, recos, 200)
}
