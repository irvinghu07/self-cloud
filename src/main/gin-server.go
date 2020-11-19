package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func singleUploadHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err == nil {
		filename := file.Filename
		cwd, err := os.Getwd()
		dir := "/resources/"
		if err == nil {
			err := c.SaveUploadedFile(file, cwd+dir+filename)
			if err != nil {
				c.JSON(http.StatusBadRequest, fmt.Sprintf("'%s' upload failed:%v", filename, err))
			} else {
				c.JSON(http.StatusOK, fmt.Sprintf("'%s' uploaded!", filename))
			}
		}
	} else {
		log.Fatal(err)
	}
}

func multiUploadHandler(context *gin.Context) {

}

func main() {
	routes := gin.Default()
	routes.MaxMultipartMemory = 8 << 20
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:8080"}
	uploadGroup := routes.Group("/upload")
	{
		uploadGroup.POST("/single", singleUploadHandler)
		uploadGroup.POST("/multi", multiUploadHandler)
	}
	err := routes.Run(":7777")
	if err != nil {
		log.Fatal("gin startup")
	}
}
