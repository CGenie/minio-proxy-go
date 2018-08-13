package main

import (
	"fmt"
	"io"
  "log"
	"net/http"
	"net/url"
	"os"
	//"time"

  "github.com/gin-gonic/gin"
  "github.com/minio/minio-go"
)


const DEFAULT_PRESIGNED_EXPIRY = 60  // in seconds


func SetupRouter() *gin.Engine {
  router := gin.Default()

	router.GET("/", Hello)
	router.GET("/download/*path", DownloadPath)
	router.GET("/download-thumbnail/:securedOrMedia/*path", DownloadThumbnailPath)

	v1 := router.Group("api/v1")
	{
		v1.GET("/hello", Hello)
	}

  return router
}

func Hello(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": "hello"})
}

func DownloadPath(c *gin.Context) {
  //c.JSON(200, gin.H{"ok": "Welcome to Chicago!"})
  // curl -i http://localhost:8080/api/v1/Instructions
	path := c.Param("path")
	if path[0] == '/' {
		path = path[1:]
	}

	ServeFile(c, path)
}

func DownloadThumbnailPath(c *gin.Context) {
	securedOrMedia := c.Param("securedOrMedia")
	path := c.Param("path")
	if path[0] == '/' {
		path = path[1:]
	}

	thumbnailPath := securedOrMedia + "/thumbnails/" + path

	ServeFile(c, thumbnailPath)
}

func ServeFile(c *gin.Context, path string) {
	minioClient := getMinioClient()
	bucket := getMinioBucket()

	objectsDoneCh := make(chan struct{})
	defer close(objectsDoneCh)

	objectsCh := minioClient.ListObjectsV2(bucket, path, false, objectsDoneCh)
	for object := range objectsCh {
		if object.Key == path {
			fmt.Println(object)

			object, err := minioClient.GetObject(bucket, path, minio.GetObjectOptions{})
			if err != nil {
				panic(err)
			}

			objectInfo, _ := object.Stat()
			c.Header("Content-Type", objectInfo.ContentType)
			c.Header("Content-Length", fmt.Sprintf("%v", objectInfo.Size))

			defer func() {
				if err := object.Close(); err != nil {
					panic(err)
				}
			}()

			buf := make([]byte, 1024)
			c.Stream(func(w io.Writer) bool {
				for {
					n, err := object.Read(buf)
					if err != nil && err != io.EOF {
						panic(err)
					}
					if n == 0 {
						break
					}
					if _, err := w.Write(buf[:n]); err != nil {
						panic(err)
					}
				}
				return false
			})

			/*
			presignedUrl, err := minioClient.PresignedGetObject(
				bucket,
				path,
				time.Second * DEFAULT_PRESIGNED_EXPIRY,
				nil)
			if err != nil {
				panic(err)
			}

			c.Header("X-Accel-Redirect", "/proxy/" + presignedUrl.String())

			//c.JSON(http.StatusOK, gin.H{"path": path, "presignedUrl": presignedUrl})
			c.String(http.StatusOK, "")
			*/

			return
		}
	}

	//c.JSON(http.StatusOK, gin.H{"path": path})
	c.String(http.StatusNotFound, fmt.Sprintf("%v: Not found", path))
}

func getMinioClient() *minio.Client {
	url_ := os.Getenv("MINIO_URL")
	u, err := url.Parse(url_)
	if err != nil {
		panic(err)
	}
	endpoint := u.Host
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	if accessKey == "" {
		panic("No access key provided")
	}
	fmt.Println(accessKey)
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	if secretKey == "" {
		panic("No secret key provided")
	}
	region := os.Getenv("MINIO_REGION")
	useSSL := (u.Scheme == "https")

	// Initialize minio client object.
	minioClient, err := minio.NewWithRegion(endpoint, accessKey, secretKey, useSSL, region)
	if err != nil {
		panic(err)
	}

	log.Printf("%#v\n", minioClient) // minioClient is now setup

	return minioClient
}

func getMinioBucket() string {
	bucket := os.Getenv("MINIO_ITALAMO_BUCKET")
	if bucket == "" {
		panic("No bucket provided")
	}
	return bucket
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}

  router := SetupRouter()
  router.Run(fmt.Sprintf(":%v", port))
}
