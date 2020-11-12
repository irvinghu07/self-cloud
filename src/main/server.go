package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type UploadInfo struct {
	Original   string
	Processed  string
	Path       string
	ResourceID string
}

const (
	uploadUrl    = "/upload"
	downloadUrl  = "/"
	port         = 7777
	mediaMaxSize = 500 * 1024 * 1024
	resourcePath = "./resources/"
)

func init() {
	if !Exists(resourcePath) {
		syscall.Umask(0)
		err := os.Mkdir(resourcePath, os.FileMode(0766))
		if err != nil {
			log.Fatal(err.Error())
		}
	}
}

func Exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func uploadHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Add("Access-Control-Allow-Method", "POST,GET")
	w.Header().Set("content-type", "application/json")
	contentType := req.Header.Get("content-type")
	contentLen := req.ContentLength

	log.Printf("uploadHandler content-type:%s\ncontent-length:%d\n", contentType, contentLen)
	if !strings.Contains(contentType, "multipart/form-data") {
		http.Error(w, "please upload media"+contentType, http.StatusForbidden)
		log.Println("please upload media" + contentType)
		return
	}

	err := req.ParseMultipartForm(mediaMaxSize)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}
	if len(req.MultipartForm.File) == 0 {
		http.Error(w, "invalid media size 0 bytes", http.StatusBadRequest)
		log.Println("invalid media size 0 bytes")
		return
	}

	for name, files := range req.MultipartForm.File {
		log.Printf("Recieving file with name=%s\n", name)
		if len(files) != 1 {
			http.Error(w, "please upload file one at a time", http.StatusBadRequest)
			log.Println("please upload file one at a time")
			return
		}
		if name == "" {
			http.Error(w, "file name error", http.StatusBadRequest)
			log.Println("file name error")
			return
		}

		for _, f := range files {
			fileResource, err := f.Open()
			if err != nil {
				w.Write([]byte(fmt.Sprintf("unknown error,fileName=%s,fileSize=%d,err:%s", f.Filename, f.Size, err.Error())))
				log.Println(err.Error())
				return
			}

			resourceName := fmt.Sprintf("%v.%v", time.Now().Nanosecond(), strings.Split(f.Filename, ".")[1])
			path := fmt.Sprintf("%v%v", resourcePath, resourceName)
			log.Printf("about to save file in Path: %v\n", path)
			fileSaveDestination, err := os.Create(path)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				log.Println(err.Error())
				return
			}
			_, err = io.Copy(fileSaveDestination, fileResource)
			if err != nil {
				http.Error(w, "saving file failed: "+err.Error(), http.StatusInternalServerError)
				log.Println("saving file failed: " + err.Error())
				return
			}
			err = fileSaveDestination.Close()
			if err != nil {
				http.Error(w, "saving file failed: close save file", http.StatusInternalServerError)
				log.Println("saving file failed: close save file")
				return
			}
			log.Printf("successful uploaded,Original fileName=%s, Processed fileName%s,savePath=%s \n", f.Filename, resourceName, path)
			resourceID := url.QueryEscape(resourceName)
			respRaw := UploadInfo{Original: f.Filename, Processed: resourceName, Path: path, ResourceID: resourceID}
			log.Printf("generating response JSON:%v\n", respRaw)
			resp, err := json.Marshal(respRaw)
			if err != nil {
				http.Error(w, "generating response failed", http.StatusInternalServerError)
				log.Println("generating response failed")
				return
			}
			_, err = w.Write(resp)
		}
	}
}

func getContentType(fileName string) (string, string) {
	arr := strings.Split(fileName, ".")
	var contentType string
	var extension string
	// see: https://tool.oschina.net/commons/
	if len(arr) >= 2 {
		extension = arr[len(arr)-1]
		switch extension {
		case "jpeg", "jpe", "jpg":
			log.Println("Inside")
			contentType = "image/jpeg"
		case "png":
			contentType = "image/png"
		case "gif":
			contentType = "image/gif"
		case "mp4":
			contentType = "video/mpeg4"
		case "mp3":
			contentType = "audio/mp3"
		case "wav":
			contentType = "audio/wav"
		case "pdf":
			contentType = "application/pdf"
		case "doc", "":
			contentType = "application/msword"
		default:
			contentType = ""
		}
		return extension, contentType
	}
	// .*（ 二进制流，不知道下载文件类型）
	contentType = "application/octet-stream"
	return extension, contentType
}

func downloadHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("content-type", "application/json")
	log.Println("Request URI: " + req.RequestURI)
	filename := req.RequestURI[1:]
	log.Printf("filename:%v\n", filename)
	enEscapeUrl, err := url.QueryUnescape(filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	f, err := os.Open(resourcePath + enEscapeUrl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	info, err := f.Stat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, contentType := getContentType(filename)
	log.Printf("Fetching media: %v with contentType:%v", filename, contentType)
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	//w.Header().Set("Content-Type", http.DetectContentType(fileHeader))
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))

	_, err = f.Seek(0, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = io.Copy(w, f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	log.Printf("linsten on :%v\n", port)
	http.HandleFunc(uploadUrl, uploadHandler)
	http.HandleFunc(downloadUrl, downloadHandler)
	err := http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
	if err != nil {
		panic(err)
	}
}
