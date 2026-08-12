package server

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"disapp/internal/model"
)

func TestUploadVersion(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a"}
	s.DB.Create(&app)
	ch := model.Channel{AppID: app.ID, Name: "test"}
	s.DB.Create(&ch)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mw.WriteField("app_id", itoa(app.ID))
	mw.WriteField("channel_id", itoa(ch.ID))
	mw.WriteField("version_name", "1.2.3")
	mw.WriteField("version_code", "123")
	mw.WriteField("changelog", "修复 bug")
	mw.WriteField("access_mode", "public")
	fw, _ := mw.CreateFormFile("file", "app.apk")
	fw.Write([]byte("fake-apk-bytes"))
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/versions", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	s.UploadVersion(w, req)

	var res struct {
		Code int `json:"code"`
		Data struct {
			ID           int64  `json:"id"`
			VersionName  string `json:"version_name"`
			FileSize     int64  `json:"file_size"`
			SHA256       string `json:"sha256"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 {
		t.Fatalf("res = %s", w.Body.String())
	}
	if res.Data.VersionName != "1.2.3" || res.Data.FileSize != int64(len("fake-apk-bytes")) {
		t.Fatalf("data = %+v", res.Data)
	}
	if res.Data.SHA256 == "" {
		t.Fatal("sha256 missing")
	}

	// Verify file was actually written to local storage
	var v model.Version
	s.DB.Last(&v)
	rc, err := s.Storage.Open(nil, itoa(app.ID)+"/"+itoa(v.ID)+"/app.apk")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "fake-apk-bytes" {
		t.Fatalf("stored = %q", data)
	}
}

func TestUploadVersionPassword(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a"}
	s.DB.Create(&app)
	ch := model.Channel{AppID: app.ID, Name: "test"}
	s.DB.Create(&ch)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mw.WriteField("app_id", itoa(app.ID))
	mw.WriteField("channel_id", itoa(ch.ID))
	mw.WriteField("version_name", "1.0")
	mw.WriteField("version_code", "10")
	mw.WriteField("access_mode", "password")
	mw.WriteField("password", "secret")
	fw, _ := mw.CreateFormFile("file", "x.apk")
	fw.Write([]byte("data"))
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/versions", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	s.UploadVersion(w, req)

	var res struct {
		Code int `json:"code"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 {
		t.Fatalf("res = %s", w.Body.String())
	}

	var v model.Version
	s.DB.Last(&v)
	if v.PasswordHash == "" {
		t.Fatal("password hash missing")
	}
}

func TestUpdateVersionToggleDisabled(t *testing.T) {
	s := testServer(t)
	v := model.Version{
		AppID: 1, VersionName: "1.0", VersionCode: 1, AccessMode: model.AccessPublic,
		Enabled: true, StorageKey: "1/2/a.apk",
	}
	s.DB.Create(&v)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/versions/"+itoa(v.ID), strings.NewReader(`{"enabled":false,"changelog":"下架"}`))
	req.SetPathValue("id", itoa(v.ID))
	w := httptest.NewRecorder()
	s.UpdateVersion(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d", w.Code)
	}
	var reload model.Version
	s.DB.First(&reload, v.ID)
	if reload.Enabled || reload.Changelog != "下架" {
		t.Fatalf("reload = %+v", reload)
	}
}

func TestDeleteVersion(t *testing.T) {
	s := testServer(t)
	v := model.Version{
		AppID: 1, VersionName: "1.0", VersionCode: 1, AccessMode: model.AccessPublic,
		Enabled: true, StorageKey: "1/2/a.apk",
	}
	s.DB.Create(&v)
	s.Storage.Save(nil, "1/2/a.apk", strings.NewReader("x"))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/versions/"+itoa(v.ID)+"?delete_file=true", nil)
	req.SetPathValue("id", itoa(v.ID))
	w := httptest.NewRecorder()
	s.DeleteVersion(w, req)
	var count int64
	s.DB.Model(&model.Version{}).Count(&count)
	if count != 0 {
		t.Fatalf("versions = %d", count)
	}
	if _, err := s.Storage.Open(nil, "1/2/a.apk"); err == nil {
		t.Fatal("file should be deleted")
	}
}

func TestVersionStats(t *testing.T) {
	s := testServer(t)
	v := model.Version{
		AppID: 1, VersionName: "1.0", VersionCode: 1, AccessMode: model.AccessPublic,
		Enabled: true, StorageKey: "1/2/a.apk", DownloadCount: 3, InstallCount: 1,
	}
	s.DB.Create(&v)
	s.DB.Create(&model.DownloadLog{VersionID: v.ID, IP: "1.2.3.4", UserAgent: "curl"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/versions/"+itoa(v.ID)+"/stats", nil)
	req.SetPathValue("id", itoa(v.ID))
	w := httptest.NewRecorder()
	s.VersionStats(w, req)
	var res struct {
		Code int `json:"code"`
		Data struct {
			DownloadCount int64              `json:"download_count"`
			InstallCount  int64              `json:"install_count"`
			Recent        []model.DownloadLog `json:"recent"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 || res.Data.DownloadCount != 3 || len(res.Data.Recent) != 1 {
		t.Fatalf("res = %s", w.Body.String())
	}
}
