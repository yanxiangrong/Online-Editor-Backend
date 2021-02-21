package main

import (
	"database/sql"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"math/rand"
	"net/http"
	"time"
)

var database *sql.DB

type EditorData struct {
	Workspace int    `json:"workspace" db:"id"`
	Content   string `json:"content" db:"content"`
	Theme     string `json:"theme" db:"theme"`
	Language  string `json:"language" db:"language"`
}

type RequestData struct {
	Workspace int `json:"workspace" db:"id"`
}

func main() {
	rand.Seed(time.Now().Unix())
	database = openDatabase()

	router := gin.Default()
	//router.StaticFS("/", http.Dir("dist"))
	v1 := router.Group("v1")
	{
		v1.POST("create", create)
		v1.POST("workspace", editorData)
		v1.OPTIONS("workspace", func(context *gin.Context) {
			context.Status(http.StatusOK)
		})
		v1.POST("upload", upload)
		v1.OPTIONS("upload", func(context *gin.Context) {
			context.Status(http.StatusOK)
		})
	}
	err := router.Run(":9527")
	if err != nil {
		log.Fatalln(err.Error())
	}
	closeDatabase(database)
}

func create(ctx *gin.Context) {
	var workspace int
	workspace = rand.Intn(900000) + 100000
	_, err := database.Exec("insert userdata (id) values (?)", workspace)
	if err != nil {
		log.Fatalln(err.Error())
	}
	ctx.JSON(http.StatusOK, gin.H{"result": 0, "workspace": workspace})
}

func editorData(ctx *gin.Context) {
	var data EditorData
	var request RequestData
	err := ctx.Bind(&request)
	if err != nil {
		log.Fatalln(err)
	}
	rows, err := database.Query("select id, content, theme, `language` from userdata where id=?", request.Workspace)
	if err != nil {
		log.Fatalln(err.Error())
	}
	err = rows.Scan(&data.Workspace, &data.Content, &data.Theme, &data.Language)
	if err != nil {
		log.Fatalln(err.Error())
	}
	err = rows.Close()
	if err != nil {
		log.Fatalln(err.Error())
	}
	ctx.JSON(http.StatusOK, gin.H{"result": 0, "data": data})
}

func upload(ctx *gin.Context) {
	var data EditorData
	err := ctx.Bind(&data)
	if err != nil {
		log.Println(err)
	}
	_, err = database.Exec("update userdata set content=?, theme=?, `language`=? where id=?", data.Content, data.Theme, data.Language, data.Workspace)
	if err != nil {
		log.Fatalln(err.Error())
	}
	ctx.JSON(http.StatusOK, gin.H{"result": 0})
}

func openDatabase() *sql.DB {
	var db, err = sql.Open("mysql", "root:passwd@tcp(127.0.0.1:3306)/mytest?charset=utf8")
	if err != nil {
		log.Fatalln(err.Error())
	}
	return db
}

func closeDatabase(database *sql.DB) {
	var err = database.Close()
	if err != nil {
		log.Fatalln(err.Error())
	}
}
