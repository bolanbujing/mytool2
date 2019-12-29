package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/astaxie/beego/logs"
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/xorm"
)

type HlsSource struct {
	ID         int64     `xorm:"pk autoincr" json:"-"`
	Url        string    `xorm:"NOT NULL" json:"url"`
	Language   string    `xorm:"NULL" json:"language"`
	Country    string    `xorm:"NULL" json:"country"`
	Category   string    `xorm:"NULL" json:"category"`
	Title      string    `xorm:"NULL" json:"Title"`
	TvgId      string    `xorm:"NULL" json:"tvg-id"`
	TvgName    string    `xorm:"NULL" json:"tvg-name"`
	TvgLan     string    `xorm:"NULL" json:"tvg-language"`
	TvgLogo    string    `xorm:"NULL" json:"tvg-logo,omitempty"`
	TvgCountry string    `xorm:"NULL" json:"tvg-country,omitempty"`
	TvgUrl     string    `xorm:"NULL" json:"tvg-url,omitempty"`
	GroupTitle string    `xorm:"NULL" json:"group-title,omitempty"`
	CreateTime time.Time `xorm:"created"`
}

const (
	TVGID      = "id"
	TVGNAME    = "name"
	TVGLAN     = "language"
	TVGLOGO    = "logo"
	TVGCOUNTRY = "country"
	TVGURL     = "url"
	GROUPTITLE = "title"
)

var engine *xorm.Engine

var queue = make(chan *HlsSource, 1000)

func (hls *HlsSource) ParseTag(arg [][]string) {
	for _, v := range arg {
		if len(v) != 3 {
			logs.Error(arg, " , v =", v, " , len is error")
			continue
		}

		key := strings.TrimSpace(v[1])
		value := strings.TrimSpace(v[2])

		switch {
		case key == TVGID:
			hls.TvgId = value
		case key == TVGNAME:
			hls.TvgName = value
		case key == TVGLAN:
			hls.TvgLan = value
		case key == TVGLOGO:
			hls.TvgLogo = value
		case key == TVGCOUNTRY:
			hls.TvgCountry = value
		case key == TVGURL:
			hls.TvgUrl = value
		case key == GROUPTITLE:
			hls.GroupTitle = value
		default:
			logs.Info("unknow tag = ", key)
		}
	}
}
func ParseHlsSourse(Dimension string, n int) {
	fi, err := os.Open(Dimension)
	if err != nil {
		logs.Error("open ", Dimension, " Error: %s", err)
		return
	}
	defer fi.Close()

	br := bufio.NewReader(fi)
	for {
		a, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
		line := string(a)
		fmt.Println(line)
		arr := strings.Split(line, "\t")
		if len(arr) != 3 && len(arr) != 4 {
			logs.Error("arr = ", arr, " , the size is error, len = ", len(arr))
			continue
		}
		res, err := http.Get(arr[2])
		if err != nil {
			logs.Error("http get arr = ", arr[2], " failed, err = ", err)
			continue
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		var resBody string
		resBody = string(body)

		line2 := strings.Split(resBody, "\n")
		line2 = line2[:len(line2)-1]
		//logs.Info(Dimension, ", ", line2)
		if len(line2) <= 1 {
			logs.Error(Dimension, " , ", arr[2], " response is empty")
			continue
		}
		for index := 1; index < len(line2); {
			x := strings.Split(line2[index], ",")
			li := x[0]
			exp := regexp.MustCompile(`tvg-(.*?)=\"(.*?)\" `)
			result := exp.FindAllStringSubmatch(li, -1)
			fmt.Println(result)
			var hls = new(HlsSource)
			hls.ParseTag(result)
			if n == 1 {
				hls.Category = strings.ToLower(arr[0])
			} else if n == 2 {
				hls.Country = strings.ToLower(arr[0])
			} else if n == 3 {
				hls.Language = strings.ToLower(arr[0])
			} else {
				logs.Error("unknow")
			}

			hls.Title = strings.Trim(strings.TrimSpace(x[1]), "\n")
			if index+1 < len(line2) {
				hls.Url = strings.Trim(line2[index+1], "\n")
			}
			queue <- hls
			index += 2
		}
	}
}

func WriteDbp() {
	for {
		hls := <-queue
		_, err := engine.Insert(hls)
		if err != nil {
			logs.Error("insert fail, hls = ", *hls)
		}
	}
}
func main() {
	logs.SetLogger(logs.AdapterFile, `{"filename":"hls.log","level":7,"maxlines":0,"maxsize":0,"daily":true,"maxdays":10,"color":true}`)
	connStr := fmt.Sprintf("%s:%s@tcp(%s)/%s%scharset=utf8&parseTime=true", "root", "123", "192.168.111.128:3306", "hls", "?")
	var err error
	engine, err = xorm.NewEngine("mysql", connStr)
	if err != nil {
		logs.Error("connect mysql fail , err = ", err)
		return
	}
	err = engine.Sync2(new(HlsSource))
	if err != nil {
		logs.Error("mysql sync HlsSource fail , err = ", err)
		return
	}
	go WriteDbp()
	//go ParseHlsSourse("./Category.txt", 1)
	go ParseHlsSourse("Country.txt", 2)
	go ParseHlsSourse("Language.txt", 3)
	select {}
}
