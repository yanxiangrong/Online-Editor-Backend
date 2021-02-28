package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/robfig/cron"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

var DataBase *sql.DB
var IsRunningCode = false
var watchDog WatchDog

const RunCodeDir = "./run/"

var MyDBConfig DatabaseConfig

var dayWorkData WorkData

type WorkData struct {
	Views   int `json:"views"`
	Uploads int `json:"uploads"`
	Runs    int `json:"runs"`
	Creates int `json:"creates"`
}

type DatabaseConfig struct {
	Username string
	Password string
	Address  string
	Port     string
	DBName   string
}

func (receiver *DatabaseConfig) ToString() string {
	return fmt.Sprintf("{Username: %s, Password: %s, Address: %s, Port: %s, DBName: %s}",
		receiver.Username, receiver.Password, receiver.Address, receiver.Port, receiver.DBName)
}

type WatchDog struct {
	timer *time.Timer
}

func (receiver *WatchDog) Init(d time.Duration) {
	if receiver.timer == nil {
		receiver.timer = time.NewTimer(d)
	} else {
		receiver.timer.Reset(d)
	}
	go func() {
		<-receiver.timer.C
		log.Fatalln("看门狗程序退出进程")
	}()
}

func (receiver *WatchDog) Stop() {
	log.Println("Run in here")
	if !receiver.timer.Stop() {
		<-receiver.timer.C
	}
}

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
	MyDBConfig = initDBConfig()
	DataBase = openDatabase()
	isPathExist, err := PathExists(RunCodeDir)
	if err != nil {
		log.Println(err.Error())
	}
	if !isPathExist {
		err = os.Mkdir(RunCodeDir, os.ModePerm)
		if err != nil {
			log.Println(err.Error())
		}
	}
	creatCleaner()

	router := gin.Default()
	//router.StaticFS("/", http.Dir("dist"))
	v1 := router.Group("v1")
	{
		v1.POST("create", func(c *gin.Context) {
			dayWorkData.Creates++
			create(c)
		})
		v1.GET("test", func(context *gin.Context) {
			context.String(http.StatusOK, "Test")
		})
		v1.POST("workspace", func(c *gin.Context) {
			dayWorkData.Views++
			editorData(c)
		})
		v1.OPTIONS("workspace", func(context *gin.Context) {
			context.Status(http.StatusOK)
		})
		v1.POST("upload", func(c *gin.Context) {
			dayWorkData.Uploads++
			upload(c)
		})
		v1.OPTIONS("upload", func(context *gin.Context) {
			context.Status(http.StatusOK)
		})
		v1.POST("run", func(c *gin.Context) {
			dayWorkData.Runs++
			editorRun(c)
		})
		v1.OPTIONS("run", func(context *gin.Context) {
			context.Status(http.StatusOK)
		})
		v1.GET("info", DayWorkData)
		v1.OPTIONS("info", func(context *gin.Context) {
			context.Status(http.StatusOK)
		})
	}
	err = router.Run(":9527")
	if err != nil {
		log.Println(err.Error())
	}
	closeDatabase(DataBase)
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func initDBConfig() DatabaseConfig {
	config := DatabaseConfig{
		os.Getenv("DB_USERNAME"),
		os.Getenv("DB_PASSWD"),
		os.Getenv("DB_ADDR"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_DBNAME"),
	}
	log.Println(config.ToString())
	return config
}

func create(ctx *gin.Context) {
	var workspace int
	for true {
		workspace = rand.Intn(900000) + 100000
		_, err := DataBase.Exec("insert userdata (id) values (?)", workspace)
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
	row := DataBase.QueryRow("select id, content, theme, `language` from userdata where id=?", request.Workspace)
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
	_, err = DataBase.Exec("update userdata set content=?, theme=?, `language`=? where id=?", data.Content, data.Theme, data.Language, data.Workspace)
	if err != nil {
		log.Println(err.Error())
		ctx.JSON(http.StatusOK, gin.H{"result": -1, "content": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"result": 0})
}

func DayWorkData(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"result": 0, "data": dayWorkData})
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
	row := DataBase.QueryRow("select id, content, theme, `language` from userdata where id=?", request.Workspace)
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

	if IsRunningCode {
		ctx.JSON(http.StatusOK, gin.H{"result": -1, "content": "当前已有代码在运行"})
		return
	} else {
		IsRunningCode = true
	}
	watchDog.Init(time.Minute)
	defer func() {
		watchDog.Stop()
		IsRunningCode = false
	}()

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
	case "python":
		result, duration, err := runCodePython(data.Content, request.InputString)
		if err != nil {
			log.Println(err.Error())
			ctx.JSON(http.StatusOK, gin.H{"result": -1, "content": err.Error(), "duration": duration})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"result": 0, "output": result, "duration": duration})
	case "java":
		result, duration, err := runCodeJava(data.Content, request.InputString)
		if err != nil {
			log.Println(err.Error())
			ctx.JSON(http.StatusOK, gin.H{"result": -1, "content": err.Error(), "duration": duration})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"result": 0, "output": result, "duration": duration})
	case "go":
		result, duration, err := runCodeGo(data.Content, request.InputString)
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
	var db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8", MyDBConfig.Username, MyDBConfig.Password, MyDBConfig.Address, MyDBConfig.Port, MyDBConfig.DBName))
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
	file, err := os.Create(RunCodeDir + "main.c")
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
	cmd := exec.CommandContext(ctx, "gcc", RunCodeDir+"main.c", "-o", RunCodeDir+"main")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		retString += "编译错误\n"
		retString += "--------------------\n"
		retString += fmt.Sprintf("%s\n", stdoutStderr)
		retString += "--------------------\n"
		retString += err.Error() + "\n"
		err = os.Remove(RunCodeDir + "main.c")
		if err != nil {
			log.Println(err.Error())
			return retString, 0, err
		}
		return retString, 0, nil
	}

	err = os.Remove(RunCodeDir + "main.c")
	if err != nil {
		log.Println(err.Error())
		return "", 0, err
	}

	cmd = exec.CommandContext(ctx, RunCodeDir+"main")
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
	err = os.Remove(RunCodeDir + "main")
	if err != nil {
		log.Println(err.Error())
		return retString, int(duration.Milliseconds()), err
	}
	retString += fmt.Sprintf("%s\n", stdoutStderr)
	return retString, int(duration.Milliseconds()), nil
}

func runCodeCpp(code string, input string) (string, int, error) {
	var retString string
	file, err := os.Create(RunCodeDir + "main.cpp")
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
	cmd := exec.CommandContext(ctx, "g++", RunCodeDir+"main.cpp", "-o", RunCodeDir+"main")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		retString += "编译错误\n"
		retString += "----------------\n"
		retString += fmt.Sprintf("%s\n", stdoutStderr)
		retString += "----------------\n"
		retString += err.Error() + "\n"
		err = os.Remove(RunCodeDir + "main.cpp")
		if err != nil {
			log.Println(err.Error())
			return retString, 0, err
		}
		return retString, 0, nil
	}

	err = os.Remove(RunCodeDir + "main.cpp")
	if err != nil {
		log.Println(err.Error())
		return "", 0, err
	}

	cmd = exec.CommandContext(ctx, RunCodeDir+"main")
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
	err = os.Remove(RunCodeDir + "main")
	if err != nil {
		log.Println(err.Error())
		return retString, int(duration.Milliseconds()), err
	}
	retString += fmt.Sprintf("%s\n", stdoutStderr)
	return retString, int(duration.Milliseconds()), nil
}

func runCodePython(code string, input string) (string, int, error) {
	var retString string
	file, err := os.Create(RunCodeDir + "main.py")
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

	cmd := exec.CommandContext(ctx, "python3", RunCodeDir+"main.py")
	cmd.Stdin = strings.NewReader(input)
	time1 := time.Now()
	stdoutStderr, err := cmd.CombinedOutput()
	duration := time.Since(time1)
	if err != nil {
		retString += "运行错误\n"
		retString += "----------------\n"
		retString += fmt.Sprintf("%s\n", stdoutStderr)
		retString += "----------------\n"
		retString += err.Error() + "\n"
		return retString, int(duration.Milliseconds()), nil
	}
	err = os.Remove(RunCodeDir + "main.py")
	if err != nil {
		log.Println(err.Error())
		return retString, int(duration.Milliseconds()), err
	}
	retString += fmt.Sprintf("%s\n", stdoutStderr)
	return retString, int(duration.Milliseconds()), nil
}

func runCodeJava(code string, input string) (string, int, error) {
	var retString string
	file, err := os.Create(RunCodeDir + "Main.java")
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
	cmd := exec.CommandContext(ctx, "javac", RunCodeDir+"Main.java")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		retString += "编译错误\n"
		retString += "----------------\n"
		retString += fmt.Sprintf("%s\n", stdoutStderr)
		retString += "----------------\n"
		retString += err.Error() + "\n"
		err = os.Remove(RunCodeDir + "Main.java")
		if err != nil {
			log.Println(err.Error())
			return retString, 0, err
		}
		return retString, 0, nil
	}

	err = os.Remove(RunCodeDir + "Main.java")
	if err != nil {
		log.Println(err.Error())
		return "", 0, err
	}

	cmd = exec.CommandContext(ctx, "java", "-classpath", RunCodeDir, "Main")
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
	err = os.Remove(RunCodeDir + "Main.class")
	if err != nil {
		log.Println(err.Error())
		return retString, int(duration.Milliseconds()), err
	}
	retString += fmt.Sprintf("%s\n", stdoutStderr)
	return retString, int(duration.Milliseconds()), nil
}

func runCodeGo(code string, input string) (string, int, error) {
	var retString string
	file, err := os.Create(RunCodeDir + "main.go")
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
	cmd := exec.CommandContext(ctx, "go", "build", "-o", RunCodeDir+"main", RunCodeDir+"main.go")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		retString += "编译错误\n"
		retString += "----------------\n"
		retString += fmt.Sprintf("%s\n", stdoutStderr)
		retString += "----------------\n"
		retString += err.Error() + "\n"
		err = os.Remove(RunCodeDir + "main.go")
		if err != nil {
			log.Println(err.Error())
			return retString, 0, err
		}
		return retString, 0, nil
	}

	err = os.Remove(RunCodeDir + "main.go")
	if err != nil {
		log.Println(err.Error())
		return "", 0, err
	}

	cmd = exec.CommandContext(ctx, RunCodeDir+"main")
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
	err = os.Remove(RunCodeDir + "main")
	if err != nil {
		log.Println(err.Error())
		return retString, int(duration.Milliseconds()), err
	}
	retString += fmt.Sprintf("%s\n", stdoutStderr)
	return retString, int(duration.Milliseconds()), nil
}

func creatCleaner() {
	myCron := cron.New()
	err := myCron.AddFunc("0 0 0 * * *", func() {
		dayWorkData.Runs = 0
		dayWorkData.Uploads = 0
		dayWorkData.Views = 0
	})
	if err != nil {
		log.Fatalln(err.Error())
	}
}
