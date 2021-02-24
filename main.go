package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

var database *sql.DB
var isRunningCode = false

type EditorData struct {
	Workspace int    `json:"workspace" db:"id"`
	Content   string `json:"content" db:"content"`
	Theme     string `json:"theme" db:"theme"`
	Language  string `json:"language" db:"language"`
}

type RequestData struct {
	Workspace int `json:"workspace" db:"id"`
}

type RequestRunData struct {
	Workspace   int    `json:"workspace" db:"id"`
	InputString string `json:"input_string" db:"input_string"`
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
		v1.POST("run", editorRun)
		v1.OPTIONS("run", func(context *gin.Context) {
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

func editorRun(ctx *gin.Context) {
	var data EditorData
	var request RequestRunData
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

	if isRunningCode {
		ctx.JSON(http.StatusOK, gin.H{"result": -1, "content": "当前已有代码在运行"})
		return
	} else {
		isRunningCode = true
	}
	defer func() { isRunningCode = false }()

	switch data.Language {
	case "c":
		result, duration, err := runCodeC(data.Content, request.InputString)
		if err != nil {
			log.Println(err.Error())
			ctx.JSON(http.StatusOK, gin.H{"result": -1, "content": err.Error(), "duration": duration})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"result": 0, "output": result, "duration": duration})
	case "cpp":
		result, duration, err := runCodeCpp(data.Content, request.InputString)
		if err != nil {
			log.Println(err.Error())
			ctx.JSON(http.StatusOK, gin.H{"result": -1, "content": err.Error(), "duration": duration})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"result": 0, "output": result, "duration": duration})
	default:
		ctx.JSON(http.StatusOK, gin.H{"result": -1, "content": "目前还不支持运行该语言"})
	}
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

func runCodeC(code string, input string) (string, int, error) {
	var retString string
	file, err := os.Create("./a.c")
	if err != nil {
		return "", 0, err
	}
	_, err = file.WriteString(code)
	if err != nil {
		return "", 0, err
	}
	err = file.Close()
	if err != nil {
		return "", 0, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gcc", "./a.c", "-o", "./a.exe")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		retString += "编译错误\n"
		retString += "--------------------\n"
		retString += fmt.Sprintf("%s\n", stdoutStderr)
		retString += "--------------------\n"
		retString += err.Error() + "\n"
		err = os.Remove("./a.c")
		if err != nil {
			log.Println(err.Error())
			return retString, 0, err
		}
		return retString, 0, nil
	}

	err = os.Remove("./a.c")
	if err != nil {
		log.Println(err.Error())
		return "", 0, err
	}

	cmd = exec.CommandContext(ctx, "./a.exe")
	cmd.Stdin = strings.NewReader(input)
	time1 := time.Now()
	stdoutStderr, err = cmd.CombinedOutput()
	duration := time.Since(time1)
	if err != nil {
		retString += "运行错误\n"
		retString += "----------------\n"
		retString += fmt.Sprintf("%s\n", stdoutStderr)
		retString += "----------------\n"
		retString += err.Error() + "\n"
		return retString, int(duration.Milliseconds()), nil
	}
	err = os.Remove("./a.exe")
	if err != nil {
		log.Println(err.Error())
		return retString, int(duration.Milliseconds()), err
	}
	retString += fmt.Sprintf("%s\n", stdoutStderr)
	return retString, int(duration.Milliseconds()), nil
}

func runCodeCpp(code string, input string) (string, int, error) {
	var retString string
	file, err := os.Create("./a.cpp")
	if err != nil {
		return "", 0, err
	}
	_, err = file.WriteString(code)
	if err != nil {
		return "", 0, err
	}
	err = file.Close()
	if err != nil {
		return "", 0, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "g++", "./a.cpp", "-o", "./a.exe")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		retString += "编译错误\n"
		retString += "----------------\n"
		retString += fmt.Sprintf("%s\n", stdoutStderr)
		retString += "----------------\n"
		retString += err.Error() + "\n"
		err = os.Remove("./a.cpp")
		if err != nil {
			log.Println(err.Error())
			return retString, 0, err
		}
		return retString, 0, nil
	}

	err = os.Remove("./a.cpp")
	if err != nil {
		log.Println(err.Error())
		return "", 0, err
	}

	cmd = exec.CommandContext(ctx, "./a.exe")
	cmd.Stdin = strings.NewReader(input)
	time1 := time.Now()
	stdoutStderr, err = cmd.CombinedOutput()
	duration := time.Since(time1)
	if err != nil {
		retString += "运行错误\n"
		retString += "----------------\n"
		retString += fmt.Sprintf("%s\n", stdoutStderr)
		retString += "----------------\n"
		retString += err.Error() + "\n"
		return retString, int(duration.Milliseconds()), nil
	}
	err = os.Remove("./a.exe")
	if err != nil {
		log.Println(err.Error())
		return retString, int(duration.Milliseconds()), err
	}
	retString += fmt.Sprintf("%s\n", stdoutStderr)
	return retString, int(duration.Milliseconds()), nil
}
