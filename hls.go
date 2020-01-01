package main

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"

	gintemplate "github.com/foolin/gin-template"
	"github.com/gin-gonic/gin"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

type channels struct {
	Name   string
	Weight int
}

type ListChannel []channels

func (s ListChannel) Len() int           { return len(s) }
func (s ListChannel) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s ListChannel) Less(i, j int) bool { return s[i].Weight > s[j].Weight }

type iptv struct {
	Name   string
	Ch     ListChannel
	Weight int
}

type ListTv []iptv

func (s ListTv) Len() int           { return len(s) }
func (s ListTv) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s ListTv) Less(i, j int) bool { return s[i].Weight > s[j].Weight }

var data ListTv
var db *sql.DB

func GetMainIndex(c *gin.Context) {
	data = data[0:0]
	rows, err := db.Query("SELECT * FROM channel_tv")
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}
	var province, chnel, weights []byte
	// Fetch rows
	for rows.Next() {
		// get RawBytes from data
		err = rows.Scan(&province, &chnel, &weights)
		if err != nil {
			panic(err.Error()) // proper error handling instead of panic in your app
		}

		fmt.Println(string(province), string(chnel), string(weights))

		bFlag := false
		for i := 0; i < len(data); i++ {
			if data[i].Name == string(province) {
				w, _ := strconv.Atoi(string(weights))
				var cs channels
				cs.Name = string(chnel)
				cs.Weight = w
				data[i].Ch = append(data[i].Ch, cs)
				data[i].Weight += w
				bFlag = true
			}
		}
		if !bFlag {
			var t iptv
			w, _ := strconv.Atoi(string(weights))
			t.Name = string(province)
			t.Weight += w
			var cs channels
			cs.Name = string(chnel)
			cs.Weight = w
			t.Ch = append(t.Ch, cs)
			data = append(data, t)
		}
		fmt.Println("-----------------------------------")
	}
	if err = rows.Err(); err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}
	fmt.Println("sort before : ", data)
	for index := 0; index < len(data); index++ {
		sort.Sort(data[index].Ch)
	}
	sort.Sort(data)
	fmt.Println("sort after : ", data)
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "bobo网",
		"data":  data,
	})
}

type Chans struct {
	Video string `uri:"video" binding:"required"`
	Title string `uri:"title" binding:"required"`
}

func GetVideo(c *gin.Context) {
	var channel Chans
	if err := c.ShouldBindUri(&channel); err != nil {
		c.JSON(400, gin.H{"msg": err})
		return
	}
	if channel.Video == "video" {
		rows, err := db.Query("SELECT title, url FROM hls_source where title=?", channel.Title)
		if err != nil {
			panic(err.Error()) // proper error handling instead of panic in your app
		}
		var title, url []byte
		// Fetch rows
		for rows.Next() {
			// get RawBytes from data
			err = rows.Scan(&title, &url)
			if err != nil {
				panic(err.Error()) // proper error handling instead of panic in your app
			}
			break
		}
		c.HTML(http.StatusOK, "video.html", gin.H{
			"title":   "bobo网",
			"channel": string(title),
			"url":     string(url),
		})
	}
}

func main() {
	r := gin.Default()
	r.HTMLRender = gintemplate.Default()
	r.GET("/", GetMainIndex)
	r.GET("/:video/:title", GetVideo)
	connStr := fmt.Sprintf("%s:%s@tcp(%s)/%s%scharset=utf8&parseTime=true", "root", "123", "192.168.146.128:3306", "hls", "?")
	var err error
	db, err = sql.Open("mysql", connStr)
	if err != nil {
		panic(err.Error()) // Just for example purpose. You should use proper error handling instead of panic
	}
	defer db.Close()
	err = db.Ping()
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}
	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
