package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/ahui2016/recoit/ibm"
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
	cacheFS = http.StripPrefix("/cache/", cacheFS)
	http.Handle("/cache/", checkLoginForFileServer(cacheFS))

	thumbFS := http.FileServer(http.Dir(cacheThumbDir))
	thumbFS = http.StripPrefix("/thumb/", thumbFS)
	http.Handle("/thumb/", checkLoginForFileServer(thumbFS))

	http.HandleFunc("/", homePage)
	http.HandleFunc("/index", checkLogin(indexPage))
	http.HandleFunc("/api/all-recos", checkLogin(getAllRecos))

	http.HandleFunc("/tag", checkLogin(tagPage))
	http.HandleFunc("/api/tag", checkLogin(getRecosByTag))

	http.HandleFunc("/add-file", checkLogin(addFilePage))
	http.HandleFunc("/api/upload-file", checkLogin(uploadHandler))
	http.HandleFunc("/api/checksum", checkLogin(checksumHandler))

	http.HandleFunc("/edit-file", checkLogin(editFilePage))
	http.HandleFunc("/api/update-file", checkLogin(updateHandler))

	http.HandleFunc("/api/reco", checkLogin(getRecoHandler))
	http.HandleFunc("/api/delete-reco", checkLogin(deleteRecoHandler))
	http.HandleFunc("/api/thumb", checkLogin(createThumbHandler))

	http.HandleFunc("/setup-cloud", checkLogin(setupCloudPage))
	http.HandleFunc("/api/setup-cloud", checkLogin(setupCloudHandler))
	http.HandleFunc("/api/check-cloud-settings", checkLogin(checkCloudSettings))

	http.HandleFunc("/create-account", createAccountPage)
	http.HandleFunc("/api/create-account", createAccountHandler)
	http.HandleFunc("/api/is-account-exist", isAccountExist)

	http.HandleFunc("/login", loginPage)
	http.HandleFunc("/api/login", loginHandler)
	http.HandleFunc("/api/check-login", checkLoginHandler)
	http.HandleFunc("/api/check-cos", checkCOS)

	http.HandleFunc("/danger/delete-first-reco", deleteFirstReco)

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
	fmt.Fprint(w, htmlFiles["add-file"])
}

func editFilePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, htmlFiles["edit-file"])
}

func tagPage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, htmlFiles["tag"])
}

func setupCloudPage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, htmlFiles["setup-cloud"])
}

func createAccountPage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, htmlFiles["create-account"])
}

func loginPage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, htmlFiles["login"])
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

	// 在 insertReco 里会添加 Reco.ID 到 Tag 数据表，并且会上传文件到 COS.
	if checkErr(w, db.InsertReco(reco, addRecoExt(reco.ID), fileContents), 500) {
		return
	}

	// 数据库操作成功，生成缓存文件（如果是图片，则顺便生成缩略图）。
	// 不可在数据库操作结束之前生成缓存文件，因为数据库操作发生错误时不应生成缓存文件。
	if checkErr(w, writeCacheFile(reco, fileContents), 500) {
		return
	}

	// 最终成功，返回新文件的 ID 给前端。
	jsonMessage(w, reco.ID, 200)
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	var maxBytes int64 = 1024 * 1024 * 3 // 3 MB
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	// 获取数据库中的 reco
	id := r.FormValue("id")
	reco, err := db.GetRecoByID(id)
	if checkErr(w, err, 500) {
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
	oldReco, err := db.GetRecoByID(id)
	if checkErr(w, err, 500) {
		return
	}

	if oldReco.EqualContent(reco) {
		jsonMessage(w, "无任何变化", 400)
		return
	}

	// 从这里开始，可以认为 reco 的内容与 oldReco 不同。
	reco.AccessCount++
	reco.AccessedAt = util.TimeNow()
	reco.UpdatedAt = reco.AccessedAt

	// TODO: update to IBM COS

	db.UpdateReco(oldReco, reco)

	// 更新缓存文件
	if reco.FileType != model.NotFile && reco.Checksum != oldReco.Checksum {
		if checkErr(w, writeCacheFile(reco, fileContents), 500) {
			return
		}
	}
}

func checksumHandler(w http.ResponseWriter, r *http.Request) {
	hashHex := r.FormValue("hashHex")
	var reco Reco
	err := db.DB.One("Checksum", hashHex, &reco)
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
	reco, err := db.GetRecoByID(id)
	if checkErr(w, err, 500) {
		return
	}
	if checkErr(w, db.AccessCountUp(id, reco.AccessCount), 500) {
		return
	}
	reco.Checksum = ""
	jsonResponse(w, reco, 200)
}

func getAllRecos(w http.ResponseWriter, r *http.Request) {
	var all []Reco
	err := db.DB.Select(q.Eq("DeletedAt", "")).OrderBy("UpdatedAt").Find(&all)
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
	checkErr(w, db.DeleteReco(id), 500)
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

		// 如果 imgPath 不存在，则从 COS 获取文件
		if util.PathIsNotExist(imgPath) {
			err := db.DownloadDecrypt(addRecoExt(id), imgPath)
			if checkErr(w, err, 500) {
				return
			}
		}

		err := util.CreateThumb(imgPath, thumbPath)
		if checkErr(w, err, 500) {
			return
		}
	}
}

func getRecosByTag(w http.ResponseWriter, r *http.Request) {
	tagName := r.FormValue("tag")
	if tagName == "" {
		jsonMessage(w, "tag's name is empty", 400)
		return
	}
	tag, err := db.GetTagByName(tagName)
	if checkErr(w, err, 500) {
		return
	}
	var recos []*Reco
	for _, id := range tag.RecoIDs {
		reco, err := db.GetRecoByID(id)
		if checkErr(w, err, 500) {
			return
		}
		reco.Checksum = ""
		recos = append(recos, reco)
	}
	jsonResponse(w, recos, 200)
}

func setupCloudHandler(w http.ResponseWriter, r *http.Request) {
	settings := ibm.Settings{
		ApiKey:            strings.TrimSpace(r.FormValue("apikey")),
		ServiceInstanceID: strings.TrimSpace(r.FormValue("serviceInstanceID")),
		ServiceEndpoint:   strings.TrimSpace(r.FormValue("endpoint")),
		BucketName:        strings.TrimSpace(r.FormValue("bucket-name")),
		// BucketLocation:    strings.TrimSpace(r.FormValue("bucket-location")),
	}
	checkErr(w, db.SetupCloud(&settings, ibmSettingsPath), 500)
}

func checkCloudSettings(w http.ResponseWriter, r *http.Request) {
	if util.PathIsExist(ibmSettingsPath) {
		jsonMsgOK(w)
	} else {
		jsonMsg404(w)
	}
}

func checkCOS(w http.ResponseWriter, r *http.Request) {
	if db.COS == nil {
		jsonMsg404(w)
	} else {
		jsonMsgOK(w)
	}
}

func checkLoginHandler(w http.ResponseWriter, r *http.Request) {
	if isLoggedIn(r) {
		jsonMessage(w, "true", 200)
	} else {
		jsonMessage(w, "false", 200)
	}
}
func isAccountExist(w http.ResponseWriter, r *http.Request) {
	if db.IsFirstRecoExist() {
		jsonMessage(w, "true", 200)
	} else {
		jsonMessage(w, "false", 200)
	}
}

func deleteFirstReco(w http.ResponseWriter, r *http.Request) {
	reco := Reco{ID: "1"}
	checkErr(w, db.DB.DeleteStruct(&reco), 500)
}

func createAccountHandler(w http.ResponseWriter, r *http.Request) {
	passphrase := r.FormValue("passphrase")
	checkErr(w, db.InsertFirstReco(passphrase), 500)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if isLoggedIn(r) {
		jsonMessage(w, "Already logged in.", 400)
		return
	}

	// db.Login 的作用是验证密码。
	passphrase := r.FormValue("passphrase")
	if err := db.Login(passphrase); err != nil {
		passwordTry++
		if checkPasswordTry(w) {
			return
		}
		jsonMessage(w, err.Error(), 400)
		return
	}

	// 如果 COS 未设置，则尝试设置，但此时忽略错误。
	if db.COS == nil {
		err := db.LoadSettings(ibmSettingsPath)
		jsonMessage(w, err.Error(), 400)
		return
	}

	// 成功登入
	passwordTry = 0
	db.Sess.Add(w, util.NewID())
}
