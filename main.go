package main

import (
	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
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
		v1.GET("test", func(context *gin.Context) {
			context.String(http.StatusOK, "Test")
		})
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
		log.Println(err.Error())
	}
	closeDatabase(database)
}

func create(ctx *gin.Context) {
	var workspace int
	for true {
		workspace = rand.Intn(900000) + 100000
		_, err := database.Exec("insert userdata (id) values (?)", workspace)
		if err != nil && err.(*mysql.MySQLError).Number == 1062 {
			continue
		}
		if err != nil {
			log.Println(err.Error())
			ctx.JSON(http.StatusOK, gin.H{"result": -1, "content": err.Error()})
			return
		}
		break
	}
	ctx.JSON(http.StatusOK, gin.H{"result": 0, "workspace": workspace})
}

func editorData(ctx *gin.Context) {
	var data EditorData
	var request RequestData
	err := ctx.Bind(&request)
	if err != nil {
		log.Println(err.Error())
		ctx.JSON(http.StatusOK, gin.H{"result": -1, "content": err.Error()})
		return
	}
	row := database.QueryRow("select id, content, theme, `language` from userdata where id=?", request.Workspace)
	err = row.Scan(&data.Workspace, &data.Content, &data.Theme, &data.Language)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusOK, gin.H{"result": -1, "content": "该工作区不存在"})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"result": -1, "content": err.Error()})
		log.Println(err.Error())
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"result": 0, "data": data})
}

func upload(ctx *gin.Context) {
	var data EditorData
	err := ctx.Bind(&data)
	if err != nil {
		log.Println(err.Error())
		ctx.JSON(http.StatusOK, gin.H{"result": -1, "content": err.Error()})
		return
	}
	if len(data.Content) > 65535 {
		ctx.JSON(http.StatusOK, gin.H{"result": -1, "content": "内容过长"})
		return
	}
	_, err = database.Exec("update userdata set content=?, theme=?, `language`=? where id=?", data.Content, data.Theme, data.Language, data.Workspace)
	if err != nil {
		log.Println(err.Error())
		ctx.JSON(http.StatusOK, gin.H{"result": -1, "content": err.Error()})
		return
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
