package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
)

var (
	sourceList []sourceType
	json               = jsoniter.ConfigCompatibleWithStandardLibrary
	source     *string = flag.String("source", "360,bing", "使用的数据源,可选(360,bing)")
	cache      *bool   = flag.Bool("cache", true, "是否在本地保存下载的壁纸，默认是")
	clear      *bool   = flag.Bool("clear", false, "清空本地下载壁纸，默认否")
)

type sourceType struct {
	url     string                                /*下载地址*/
	target  string                                /*所属源*/
	formate func() string                         /*格式化函数*/
	getFunc func(item sourceType) (string, error) /*获取图片地址函数*/
}

func init() {
	createDir()
	flag.Parse()
}

func main() {
	// 处理命令行参数
	flagParse()
	// 随机获取一个图片来源
	rand.Seed(time.Now().UnixNano())
	size, index := len(sourceList), 0
	if size == 0 {
		log.Fatalf("资源列表为空！！！")
	} else if size > 1 {
		index = rand.Intn(size - 1)
	}
	item := sourceList[index]
	// 执行对应的地址获取函数
	url, err := item.getFunc(item)
	if err != nil {
		handleError(err)
	}
	// 下载图片
	path, err := downPicture(&url)
	if err != nil {
		handleError(err)
	}
	// 设置背景图片
	e := setPicture(&path)
	if e != nil {
		handleError(err)
	}
}

func flagParse() {
	if *clear {
		os.RemoveAll("img/")
		createDir()
	}
	list := strings.Split(*source, ",")
	for _, v := range list {
		switch v {
		case "360":
			init360()
		case "bing":
			sourceList = append(sourceList, sourceType{
				url:    "https://bing.ioliu.cn/v1/rand",
				target: "bing",
				getFunc: func(item sourceType) (string, error) {
					return item.url, nil
				},
			})
		default:
			log.Fatalf("未知数据源：%s\n", v)
		}
	}
}

func init360() {
	// 先获取各分区id
	resp, err := http.Get("http://lab.mkblog.cn/wallpaper/api.php?cid=360tags")
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()
	m := make(map[string]interface{})
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&m)
	if err != nil {
		log.Println(err)
		return
	}
	if data, ok := m["data"].([]interface{}); ok {
		for _, item := range data {
			if it, ok := item.(map[string]interface{}); ok {
				if id, ok := it["id"].(string); ok {
					s := sourceType{}
					s.url = "http://wallpaper.apc.360.cn/index.php?c=WallPaper&a=getAppsByCategory&cid=" + id + "&start=%s&count=1&from=360chrome"
					s.formate = func() string {
						rand.Seed(time.Now().UnixNano())
						return strconv.Itoa(rand.Intn(128))
					}
					s.getFunc = picture360
					s.target = "360"

					sourceList = append(sourceList, s)
				}
			}
		}
	}
}

func picture360(item sourceType) (string, error) {
	// 随机获取精品图片
	picPath := fmt.Sprintf(item.url, item.formate())
	log.Printf("获取列表地址：%s", picPath)
	resp, err := http.Get(picPath)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	m := make(map[string]interface{})
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&m)
	if err != nil {
		return "", err
	}
	// 获取图片真实下载地址
	if data, ok := m["data"].([]interface{}); ok {
		if it, ok := data[0].(map[string]interface{}); ok {
			if url, ok := it["url"].(string); ok {
				url = strings.ReplaceAll(url, "bdr", "bdm")
				url = strings.ReplaceAll(url, "__85", "0_0_100")
				return url, nil
			}
		}
	}
	return "", errors.New("获取下载地址失败")
}

// 保存图片
func downPicture(url *string) (string, error) {
	u, fileName := *url, "./img/"+time.Now().Format("2006-01-02 15:04:05")+".jpg"
	log.Printf("开始下载图片，地址：%s", u)
	resp, err := http.Get(u)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	file, err := os.Create(fileName)
	if err != nil {
		return "", err
	}
	io.Copy(file, resp.Body)
	local, _ := os.Getwd()
	path := filepath.Join(local, fileName)
	log.Printf("本地缓存地址：%s\n", path)
	return path, nil
}

// 设置背景图片
func setPicture(path *string) error {
	o := runtime.GOOS
	switch o {
	case "darwin":
		cmd := exec.Command("/bin/bash", "-c", fmt.Sprintf("osascript -e 'tell application \"System Events\" to set picture of every desktop to \"%s\"'", *path))
		err := cmd.Start()
		if err != nil {
			return err
		}
		if !*cache {
			os.Remove(*path)
		}
		log.Println("set success!!!")
		return nil
	default:
		return errors.New("不支持的系统类型，请耐心等待添加")
	}
}

// 创建图片保存目录
func createDir() {
	dirName := "img"
	_, err := os.Stat(dirName)
	if err != nil && os.IsNotExist(err) {
		e := os.Mkdir(dirName, os.ModePerm)
		if e != nil {
			panic(err)
		}
	}
}

// 异常处理
func handleError(err error) {
	log.Println(err)
	panic(err)
}

// osascript -e "tell application \"Finder\" to set desktop picture to POSIX file \"/Volumes/Ramdisk/01.jpg\""
