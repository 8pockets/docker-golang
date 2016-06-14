package main

import (
	"fmt"
	"log"
	//	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/PuerkitoBio/goquery"
	"github.com/patrickmn/go-cache"
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

	w.Header().Set("Content-Type", "application/json")

	//	name := pat.Param(ctx, "name")

	//http://153.121.33.54:8881/groumet/list/?area=tokyo.A1301.A130101.R3368

	//http://stackoverflow.com/questions/15407719/in-gos-http-package-how-do-i-get-the-query-string-on-a-post-request
	q := r.URL.Query()["area"]

	//第４引数を-1にすることで対象範囲が全てになる
	area := strings.Replace(q[0], ".", "/", -1)

	param_slice := strings.Split(area, "/")

	//最後のパラメータを削除して、"/"で繋いだStringに変換
	param := strings.Join(param_slice[:3], "/")

	//変数の型を調べる
	//log.Info(reflect.TypeOf(param))
	//log.Fatal(param)

	//cache
	ca := cache.New(10*time.Minute, 30*time.Second)

	if x, found := ca.Get(param); found {
		log.Print("Cache Use !")
		fmt.Fprintf(w, "%s", x)
		return
	}

	//hatebu API request
	body := getHatebuRssFeed(param)

	//XML Parse
	//ex.) https://golang.org/pkg/encoding/xml/#example_Unmarshal
	v := TabelogXml{}
	err := xml.Unmarshal([]byte(body), &v)
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}

	var t Itemslice
	//t := Itemslice{}  //->【TBD】上の書き方との違いを調べる

	for _, bookmark := range v.Bookmarks {
		datetime, _ := time.Parse(time.RFC3339, bookmark.Date)

		title := bookmark.Title
		url := bookmark.Link
		date := datetime.Format("2006/01/00 10:00:00")
		bookmark := bookmark.Count

		//URL Validation. ( Valid url is http://tabelog.com/tokyo/A1301/A130101/13002457/ )
		matchPattern := "^http:\\/\\/tabelog\\.com\\/[a-zA-Z0-9]+\\/[a-zA-Z0-9]+\\/[a-zA-Z0-9]+\\/[a-zA-Z0-9]+(|\\/)$"
		if m, _ := regexp.MatchString(matchPattern, url); !m {
			continue
		}

		t.Items = append(t.Items, Item{Title: title, Url: url, Date: date, Bookmark: bookmark})
	}

	//お店のデータを取得(画像・星・最寄駅)
	response := getGroumetInfo(t)

	b, err_json := json.Marshal(response)

	if err_json != nil {
		fmt.Println("json err:", err_json)
	}

	//Cache Set
	ca.Set(param, b, cache.NoExpiration)

	if _, ca_result := ca.Get(param); ca_result {
		fmt.Print("Cache Correct!")
	}

	//output
	fmt.Fprintf(w, "%s", b)
}

/*
tabelogのユーザー投稿の最初画像をスクレイピング
*/
func getGroumetInfo(t Itemslice) Itemslice {

	var wg sync.WaitGroup

	runtime.GOMAXPROCS(runtime.NumCPU())
	s := time.Now()

	for i, v := range t.Items {

		request_url := v.Url

		wg.Add(1)

		go func(request_url string, i int) {

			defer wg.Done()

			image, err_request := goquery.NewDocument(request_url + "dtlphotolst/1/smp2/D-like/")

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

			log.Print(request_url + " , " + img_url + " , " + star + " , " + station)
			log.Printf("running %d goroutines", runtime.NumGoroutine())

		}(request_url, i)

	} //end t.Items for loop

	wg.Wait()

	e := time.Now().Sub(s)
	fmt.Println(e)

	return t
}

func getHatebuRssFeed(param string) []byte {

	//APi Request
	values := url.Values{}
	values.Set("mode", "rss")
	values.Add("sort", "count")
	values.Add("url", TABELOG_URL+param)

	resp, err := http.Get(HATEBU_URL + "?" + values.Encode())
	if err != nil {
		log.Fatal(err)
	}

	// 関数を抜ける際に必ずresponseをcloseするようにdeferでcloseを呼ぶ
	defer resp.Body.Close()

	// Responseの内容を使用して後続処理を行う
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return body

}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	mux := goji.NewMux()
	mux.HandleFuncC(pat.Get("/groumet/list/"), groumet)

	http.ListenAndServe(":5000", mux)
}
