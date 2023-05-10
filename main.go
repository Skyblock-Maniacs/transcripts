package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {
	log.Println("Starting server...")
	loadEnv()

	initServer()
}

func loadEnv() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal("Error loading .env file")
	}
	log.Println("Env loaded successfully")
}

func initServer() {
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
	id := c.Param("id")
	c.File("./files/" + id + ".html")
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

	err = os.WriteFile("./files/"+id+".html", extractedFileBytes, 0644)

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
