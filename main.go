package main

import (
	"fmt"
	"log"
	//	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	//"sync"
	"time"

	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/PuerkitoBio/goquery"
	//"github.com/golang/groupcache"
	"goji.io"
	"goji.io/pat"
	"golang.org/x/net/context"
)

const (
	HATEBU_URL  = "http://b.hatena.ne.jp/entrylist"
	TABELOG_URL = "http://tabelog.com/"
)

//XML struct
type TabelogXml struct {
	Bookmarks []struct {
		Title string `xml:"title"`
		Link  string `xml:"link"`
		Date  string `xml:"date"`
		Count int    `xml:"bookmarkcount"`
	} `xml:"item"`
}

//Json struct
type Item struct {
	Title    string
	Url      string
	Date     string
	Bookmark int
	Img_url  string
	Star     string
	Station  string
}
type Itemslice struct {
	Items []Item
}

func groumet(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	//URLの中の変数をとる時に使う
	//	name := pat.Param(ctx, "name")

	//if q, ok := r.URL.Query()["area"]; ok {
	//	fmt.Printf("Error * area key %#v does not exist.", q)
	//}
	q := r.URL.Query()["area"]

	//第４引数を-1にすることで対象範囲が全てになる
	area := strings.Replace(q[0], ".", "/", -1)

	param_slice := strings.Split(area, "/")

	//最初から３つ目までのパラメータだけ使用
	param := strings.Join(param_slice[:3], "/")

	//変数の型を調べる
	//log.Info(reflect.TypeOf(param))

	//hatebu API request
	i := hatebu(param)

	//お店のデータを取得(画像・星・最寄駅)
	response := tabelog(i)

	b, err_json := json.Marshal(response)
	if err_json != nil {
		fmt.Println("json err:", err_json)
	}

	//cache
	//responseCache := groupcache.NewGroup("responseCache", 64<<20, groupcache.GetterFunc(
	//	func(ctx groupcache.Context, key string, dest groupcache.Sink) error {
	//		dest.SetString(b)
	//		return nil
	//	}))

	//response := ""
	//if cache_err := responseCache.Get(nil, "response", groupcache.StringSink(&response)); cache_err != nil {
	//	fmt.Println("Cache err", cache_err)
	//}
	//if response != "" {
	//	w.Write(response)
	//	return
	//}

	//output
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func hatebu(param string) Itemslice {

	//APi Request
	values := url.Values{}
	values.Set("mode", "rss")
	values.Add("sort", "count")
	values.Add("url", TABELOG_URL+param)

	resp, err := http.Get(HATEBU_URL + "?" + values.Encode())
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	//XML Parse
	//ex.) https://golang.org/pkg/encoding/xml/#example_Unmarshal
	v := TabelogXml{}
	err = xml.Unmarshal([]byte(body), &v)
	if err != nil {
		fmt.Printf("error: %v", err)
	}

	i := Itemslice{}

	for _, bookmark := range v.Bookmarks {
		datetime, _ := time.Parse(time.RFC3339, bookmark.Date)

		title := bookmark.Title
		url := bookmark.Link
		date := datetime.Format("2006/02/00 10:00:00")
		bookmark := bookmark.Count

		//URL Validation. 時々特集記事が上がってくる( Valid url is http://tabelog.com/tokyo/A1301/A130101/13002457/ )
		matchPattern := "^http:\\/\\/tabelog\\.com\\/[a-zA-Z0-9]+\\/[a-zA-Z0-9]+\\/[a-zA-Z0-9]+\\/[a-zA-Z0-9]+(|\\/)$"
		if m, _ := regexp.MatchString(matchPattern, url); !m {
			continue
		}

		i.Items = append(i.Items, Item{Title: title, Url: url, Date: date, Bookmark: bookmark})
	}

	return i
}

//tabelogのユーザー投稿の最初画像をスクレイピング
func tabelog(t Itemslice) Itemslice {

	log.Printf("goroutine run %s", strconv.Itoa(len(t.Items)))
	s := time.Now()

	//make channel
	resultCh := make(chan error, 1)

	//context
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	ctx = context.WithValue(ctx, "Itemslice", t)
	defer cancel()

	for i, _ := range t.Items {
		go func(i int) {
			resultCh <- getTabelogData(ctx, i)
		}(i)
	}

	for i, _ := range t.Items {
		select {
		case <-resultCh:
			log.Println("done")
		case <-ctx.Done():
			log.Println(ctx.Err(), i)
		}
	}

	e := time.Now().Sub(s)
	fmt.Println(e)

	return t
}

func getTabelogData(ctx context.Context, i int) error {

	t := ctx.Value("Itemslice").(Itemslice)
	request_url := t.Items[i].Url

	image, err_request := goquery.NewDocument(request_url + "dtlphotolst/1/smp2/")

	if err_request != nil {
		fmt.Print("url scarapping failed")
	}

	//Parse HTML By goquery module
	img_url, exists := image.Find("ul.rstdtl-photo__content > li.thum-photobox .thum-photobox__img img").First().Attr("src")
	star := image.Find("div.rdheader-rating__score b.tb-rating__val span").First().Text()
	station := image.Find("div.rdheader-subinfo div.linktree__parent span").First().Text()

	if exists != true {
		fmt.Print("Not Existing Data: " + request_url)
	}

	t.Items[i].Img_url = img_url
	t.Items[i].Star = star
	t.Items[i].Station = station

	log.Printf("running %d goroutines", runtime.NumGoroutine())

	return ctx.Err()
}

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	mux := goji.NewMux()
	mux.HandleFuncC(pat.Get("/groumet/list/"), groumet)

	http.ListenAndServe(":5000", mux)
}
