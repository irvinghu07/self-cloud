package test

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
	"time"
)

type UploadInfo struct {
	Original  string
	Processed string
	Path      string
}

func setupResponse(w *http.ResponseWriter, req *http.Request) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

const (
	uploadUrl    = "/upload"
	downloadUrl  = "/download"
	port         = 7777
	mediaMaxSize = 500 * 1024 * 1024
	resourcePath = "./resources/"
)

func uploadHandler(w http.ResponseWriter, req *http.Request) {
	setupResponse(&w, req)
	contentType := req.Header.Get("content-type")
	contentLen := req.ContentLength

	log.Printf("uploadHandler content-type:%s\ncontent-length:%d\n", contentType, contentLen)
	if !strings.Contains(contentType, "multipart/form-data") {
		http.Error(w, "please upload media", http.StatusForbidden)
		log.Println("please upload media")
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
			fileSaveDestination, _ := os.Create(path)
			_, err = io.Copy(fileSaveDestination, fileResource)
			if err != nil {
				http.Error(w, "saving file failed: copy", http.StatusInternalServerError)
				log.Println("saving file failed: copy")
				return
			}
			err = fileSaveDestination.Close()
			if err != nil {
				http.Error(w, "saving file failed: close save file", http.StatusInternalServerError)
				log.Println("saving file failed: close save file")
				return
			}
			log.Printf("successful uploaded,Original fileName=%s, Processed fileName%s,savePath=%s \n", f.Filename, resourceName, path)
			respRaw := UploadInfo{Original: f.Filename, Processed: resourceName, Path: path}
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

func getContentType(fileName string) (extension, contentType string) {

	arr := strings.Split(fileName, ".")

	// see: https://tool.oschina.net/commons/
	if len(arr) >= 2 {
		extension = arr[len(arr)-1]
		switch extension {
		case "jpeg", "jpe", "jpg":
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
		}
	}
	// .*（ 二进制流，不知道下载文件类型）
	contentType = "application/octet-stream"
	return
}

func downloadHandler(w http.ResponseWriter, req *http.Request) {
	setupResponse(&w, req)
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

//func main() {
//	log.Printf("linsten on :%v\n", port)
//	http.HandleFunc(uploadUrl, uploadHandler)
//	http.HandleFunc(downloadUrl, downloadHandler)
//	err := http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
//	if err != nil {
//		panic(err)
//	}
//}
