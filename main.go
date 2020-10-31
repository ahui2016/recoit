package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/ahui2016/recoit/cloud"
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

	tempFS := http.FileServer(http.Dir(tempDir))
	tempFS = http.StripPrefix("/temp/", tempFS)
	http.Handle("/temp/", checkLoginForFileServer(tempFS))

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

	http.HandleFunc("/file", checkLogin(editFilePage))
	http.HandleFunc("/api/update-file", checkLogin(updateHandler))
	http.HandleFunc("/api/reco", checkLogin(getRecoHandler))
	http.HandleFunc("/api/delete-reco", checkLogin(deleteRecoHandler))
	http.HandleFunc("/api/create-thumb", checkLogin(createThumbHandler))
	http.HandleFunc("/api/download-file", checkLogin(downloadFile))

	http.HandleFunc("/change-box", checkLogin(changeBoxPage))
	http.HandleFunc("/api/change-box", checkLogin(changeBox))
	http.HandleFunc("/api/all-boxes", checkLogin(getAllBoxes))

	http.HandleFunc("/box", checkLogin(boxPage))
	http.HandleFunc("/api/get-box", checkLogin(getBoxHandler))
	http.HandleFunc("/api/get-recos-by-box", checkLogin(getRecosByBox))
	http.HandleFunc("/api/rename-box", checkLogin(renameBoxHandler))

	http.HandleFunc("/setup-cloud/ibm", setupIbmCosPage)
	http.HandleFunc("/api/setup-ibm-cos", setupIbmCosHandler)
	http.HandleFunc("/api/check-cloud-settings", checkCloudSettings)

	http.HandleFunc("/create-account", createAccountPage)
	http.HandleFunc("/api/create-account", createAccountHandler)
	http.HandleFunc("/api/is-account-exist", isAccountExist)

	http.HandleFunc("/login", loginPage)
	http.HandleFunc("/logout", logoutHandler)
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
		// fmt.Fprint(w, HTML["index"])
		http.Redirect(w, r, "/index", 302)
	default:
		jsonMsg404(w)
	}
}

func indexPage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, HTML["index"])
}

func addFilePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, HTML["add-file"])
}

func editFilePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, HTML["file"])
}

func changeBoxPage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, HTML["change-box"])
}

func tagPage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, HTML["tag"])
}

func boxPage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, HTML["box"])
}

func setupIbmCosPage(w http.ResponseWriter, r *http.Request) {
	if db.GCM == nil {
		fmt.Fprint(w, HTML["login"])
		return
	}
	fmt.Fprint(w, HTML["setup-ibm-cos"])
}

func createAccountPage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, HTML["create-account"])
}

func loginPage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, HTML["login"])
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {

	// 在 getFileContents 里更改了 r.Body
	fileContents, err := getFileContents(w, r)
	if checkErr(w, err, 400) {
		return
	}

	// 新建一个 Reco, 获得其 ID
	reco, err := model.NewFile(r.FormValue("file-name"))
	if checkErr(w, err, 400) {
		return
	}

	reco.Checksum = r.FormValue("checksum")
	reco.FileSize = int64(len(fileContents))

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

func getFileContents(w http.ResponseWriter, r *http.Request) ([]byte, error) {

	var maxBytes int64 = 1024 * 1024 * 3 // 3 MB
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	file, _, err := r.FormFile("file")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 将文件内容全部读入内存
	contents, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	// 根据文件内容生成 checksum 并检查其是否正确
	if util.Sha256Hex(contents) != r.FormValue("checksum") {
		return nil, errors.New("Error Checksum Unmatching")
	}
	return contents, nil
}

func updateHandler(w http.ResponseWriter, r *http.Request) {

	// 在 getFileContents 里更改了 r.Body
	fileContents, err := getFileContents(w, r)
	if err != nil && err != http.ErrMissingFile {
		jsonMessage(w, err.Error(), 500)
		return
	}

	// 获取数据库中的 reco
	id := r.FormValue("id")
	reco, err := db.GetRecoByID(id)
	if checkErr(w, err, 400) {
		return
	}

	// 为节省流量、减少出错，禁止上传相同文件。(在前端防止)
	// 本来还应该检查 checksum 的唯一性，但替换文件是低频操作，因此可以偷懒。

	// 更新 reco 的内容
	if checkErr(w, updateReco(r, fileContents, reco), 500) {
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

	if checkErr(w, db.UpdateReco(oldReco, reco, addRecoExt(id), fileContents), 500) {
		return
	}

	// 更新缓存文件
	if fileContents != nil && reco.Checksum != oldReco.Checksum {
		if checkErr(w, writeCacheFile(reco, fileContents), 500) {
			return
		}
	}
}

// updateReco updates checksum, filesize, filename, message, Box,
// links, tags of a reco
func updateReco(r *http.Request, fileContents []byte, reco *Reco) error {
	if fileContents != nil {
		reco.Checksum = util.Sha256Hex(fileContents)
		reco.FileSize = int64(len(fileContents))
	}
	if err := reco.SetFileNameType(r.FormValue("file-name")); err != nil {
		return err
	}
	reco.Message = strings.TrimSpace(r.FormValue("description"))

	// TODO: to update reco.Box

	fileLinks := []byte(r.FormValue("file-links"))
	if err := json.Unmarshal(fileLinks, &reco.Links); err != nil {
		return err
	}
	fileTags := []byte(r.FormValue("file-tags"))
	if err := json.Unmarshal(fileTags, &reco.Tags); err != nil {
		return err
	}
	return nil
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
	if checkErr(w, db.SetRecoAccessed(id, reco.AccessCount+1), 500) {
		return
	}
	jsonResponse(w, reco, 200)
}

func getAllRecos(w http.ResponseWriter, r *http.Request) {
	var all []*Reco
	err := db.DB.
		Select(q.Eq("DeletedAt", ""), q.Gt("ID", "1")).
		OrderBy("UpdatedAt").
		Find(&all)
	if checkErr(w, err, 500) {
		return
	}
	for _, reco := range all {
		reco.Checksum = ""
		if checkErr(w, boxShowTitle(reco), 500) {
			return
		}
	}
	jsonResponse(w, all, 200)
}

// 把 box.ID 转换为 box.Title 方便前端显示。
func boxShowTitle(reco *Reco) error {
	if reco.Box != "" {
		box, err := db.GetBoxByID(reco.Box)
		if err != nil {
			return err
		}
		reco.Box = box.Title
	}
	return nil
}

func getAllBoxes(w http.ResponseWriter, r *http.Request) {
	var boxes []Box
	err := db.DB.AllByIndex("UpdatedAt", &boxes, storm.Reverse())
	if checkErr(w, err, 500) {
		return
	}
	jsonResponse(w, boxes, 200)
}

func deleteRecoHandler(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	checkErr(w, db.DeleteReco(id), 500)
}

func createThumbHandler(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if checkIDEmpty(w, id) {
		return
	}

	imgPath := cacheFilePath(id)
	thumbPath := cacheThumbPath(id)

	// 验证这个文件是图片（省略，因为省略也不会出大问题）

	// 本来还要检查缩略图是否存在，但为了同时适用于别的场景
	// （比如缩略图存在，但大图不存在的情况）因此不检查缩略图是否存在。

	if util.PathIsNotExist(imgPath) {
		err := db.DownloadDecrypt(addRecoExt(id), imgPath)
		if checkErr(w, err, 500) {
			return
		}
	}
	checkErr(w, util.CreateThumb(imgPath, thumbPath), 500)
}

func getRecosByTag(w http.ResponseWriter, r *http.Request) {
	tagName := r.FormValue("tag")
	recos, err := db.GetRecosByTag(tagName)
	if checkErr(w, err, 500) {
		return
	}

	// 在返回给前端之前进行一些处理（删除不需要的，添加需要的）。
	for _, reco := range recos {
		reco.Checksum = ""
		if checkErr(w, boxShowTitle(reco), 500) {
			return
		}
	}
	jsonResponse(w, recos, 200)
}

func getRecosByBox(w http.ResponseWriter, r *http.Request) {
	boxID := r.FormValue("box-id")
	recos, err := db.GetRecosByBox(boxID)
	if checkErr(w, err, 500) {
		return
	}

	// 在返回给前端之前进行一些处理（删除不需要的，添加需要的）。
	for _, reco := range recos {
		reco.Checksum = ""
	}
	jsonResponse(w, recos, 200)
}

func setupIbmCosHandler(w http.ResponseWriter, r *http.Request) {
	if db.GCM == nil {
		jsonRequireLogin(w)
		return
	}
	settings := ibm.Settings{
		Provider:          cloud.IBM,
		ApiKey:            strings.TrimSpace(r.FormValue("apikey")),
		ServiceInstanceID: strings.TrimSpace(r.FormValue("serviceInstanceID")),
		ServiceEndpoint:   strings.TrimSpace(r.FormValue("endpoint")),
		BucketName:        strings.TrimSpace(r.FormValue("bucket-name")),
		// BucketLocation:    strings.TrimSpace(r.FormValue("bucket-location")),
	}
	checkErr(w, db.SetupIbmCos(&settings), 500)
}

func checkCloudSettings(w http.ResponseWriter, r *http.Request) {
	if util.PathIsExist(cosSettingsPath) {
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

	// 当且只当 COS 未设置时才尝试设置。
	if db.COS == nil {
		if checkErr(w, db.LoadSettings(), 500) {
			return
		}
	}

	// 成功登入
	passwordTry = 0
	db.Sess.Add(w, util.NewID())
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	db.Reset()
	db.Sess.DeleteSID(w, r)
	http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
}

// downloadFile 检查本地缓存有无该 id 的文件，如果没有就从 COS 下载。
// 最后向前端返回该文件的 url.
func downloadFile(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if checkIDEmpty(w, id) {
		return
	}

	// 如果 cache 文件夹有文件，就直接使用。
	if util.PathIsExist(cacheFilePath(id)) {
		jsonMessage(w, cacheFileURL(id), 200)
		return
	}

	// 如果 cache 文件夹找不到文件，就下载到 temp 文件夹里。
	tempFile := tempFilePath(id)
	if util.PathIsNotExist(tempFile) {
		err := db.DownloadDecrypt(addRecoExt(id), tempFile)
		if checkErr(w, err, 500) {
			return
		}
	}
	jsonMessage(w, tempFileURL(id), 200)
}

func getBoxHandler(w http.ResponseWriter, r *http.Request) {
	boxID := r.FormValue("box-id")
	box, err := db.GetBoxByID(boxID)
	if checkErr(w, err, 500) {
		return
	}
	jsonResponse(w, box, 200)
}

func changeBox(w http.ResponseWriter, r *http.Request) {
	recoID := r.FormValue("id")
	boxID := r.FormValue("box-id")
	boxTitle := strings.TrimSpace(r.FormValue("box-title"))

	if recoID == "" || boxTitle == "" {
		jsonMessage(w, "id or box-title is empty", 400)
		return
	}

	checkErr(w, db.ChangeBox(boxID, boxTitle, recoID), 500)
}

func renameBoxHandler(w http.ResponseWriter, r *http.Request) {
	boxID := r.FormValue("box-id")
	boxTitle := strings.TrimSpace(r.FormValue("box-title"))

	if boxID == "" || boxTitle == "" {
		jsonMessage(w, "box-id or box-title is empty", 400)
		return
	}

	checkErr(w, db.RenameBox(boxID, boxTitle), 500)
}
