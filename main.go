package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

var (
	client *s3.Client
)

func init() {
	log.Println("Starting server...")
	err := godotenv.Load()

	if err != nil {
		log.Fatal("Error loading .env file")
	}
	log.Println("Env loaded successfully")

	client = s3.New(s3.Options{
		AppID: "my-application/0.0.1",

		Region: os.Getenv("AWS_REGION"),
		BaseEndpoint: aws.String(os.Getenv("AWS_ENDPOINT_URL")),

		Credentials: credentials.StaticCredentialsProvider{Value: aws.Credentials{
			AccessKeyID: os.Getenv("AWS_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		}},
	})
	log.Println("S3 session created successfully")
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(cors.Default())

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"message": "Endpoint Not Found"})
	})

	r.GET("/transcripts/:id", getTranscript)
	r.POST("/transcripts", middleware(), postTranscript)
	r.DELETE("/transcripts/:id", middleware(), deleteTranscript)

	log.Println("Server started on " + os.Getenv("PORT"))
	r.Run()
}

func middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		headerToken := c.GetHeader("Authorization")
		if headerToken != os.Getenv("AUTH_TOKEN") {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func getTranscript(c *gin.Context) {
	ctx := context.Background()
	
	result, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(os.Getenv("AWS_BUCKET")),
		Key: aws.String("transcripts/" + c.Param("id") + ".html"),
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer result.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(result.Body)

	c.Data(
		http.StatusOK,
		"text/html",
		buf.Bytes(),
	)
}

func postTranscript(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error Uploading File",
		})
		return
	}
	file := form.File["file"][0]

	if !(strings.HasPrefix(file.Header.Get("content-type"), "text/html")) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File must be of type text/html"})
		return
	}

	extractedFile, err := file.Open()

	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error Uploading File",
		})
		return
	}

	id := strings.Split(uuid.New().String(), "-")[0]

	extractedFileBytes, err := io.ReadAll(extractedFile)

	extractedFile.Close()

	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error Uploading File",
		})
		return
	}

	ctx := context.Background()
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(os.Getenv("AWS_BUCKET")),
		Key: aws.String("transcripts/" + id + ".html"),
		ContentType: aws.String("text/html"),
		Body: bytes.NewReader(extractedFileBytes),
	})

	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error Uploading File",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"url": os.Getenv("URI") + "/" + id})
}

func deleteTranscript(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}
