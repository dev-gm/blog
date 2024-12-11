package main

import (
	"embed"
	"encoding/json"
	"html/template"
	"log"
	"maps"
	"net"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	
	"github.com/gomarkdown/markdown"
	mdHtml "github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/gofiber/fiber/v2/middleware/earlydata"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/helmet/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/sopa0/htmx-fiber"
)

type NavLink struct {
	Href string
	Text string
	ImageSrcDark string
	ImageSrcLight string
	ImageAltText string
}

var settings = fiber.Map{
	"NavHome": "gavin m's blog",
	"NavLinks": []NavLink{
		{
			Href: "https://codeberg.org/dev-gm",
			Text: "dev-gm",
			ImageSrcDark: "/assets/icons/codeberg.svg",
			ImageSrcLight: "/assets/icons/codeberg-blue.svg",
			ImageAltText: "cb",
		},
		{
			Href: "https://github.com/dev-gm",
			Text: "dev-gm",
			ImageSrcDark: "/assets/icons/github.svg",
			ImageSrcLight: "/assets/icons/github.svg",
			ImageAltText: "gh",
		},
	},
}

//go:embed views/*
var viewsDir embed.FS

//go:embed data/assets/*
var assetsDir embed.FS

type Article struct {
	Path string
	Url string
	Title string
	Subtitle string
	Date string
	UnixMillis int64
	HTMLBody template.HTML
	Series *Series
}

type Series struct {
	Path string
	Dirname string
	Title string
	Description string
	Date string
	UnixMillis int64
	// key is path
	PartsMap map[string]*Article
	// ordered by date
	Parts []*Article
}

var dataLock sync.RWMutex

// key is path, no series articles
var articlesMap map[string]*Article
// key is path
var seriesMap map[string]*Series

// ordered by date, include series articles
var allArticles []Article
// ordered by date
var allSeries []Series

func ServePage(c *fiber.Ctx, title string, data fiber.Map) error {
	title = "views/" + title
	data["Settings"] = &settings
	if htmx.IsHTMX(c) {
		htmx.NewResponse().Retarget("body").Write(c)
		return c.Status(200).Render(title, data, "views/nested/body")
	} else {
		return c.Status(200).Render(title, data,
			"views/nested/body", "views/nested/index")
	}
}

func ServeHome(c *fiber.Ctx) error {
	dataLock.RLock()
	defer dataLock.RUnlock()
	return ServePage(c, "home", fiber.Map{
		"AllArticles": allArticles,
		"PageTitle": "gavin m's blog",
		"PageId": "home",
	})
}

func ServeArticleHome(c *fiber.Ctx) error {
	dataLock.RLock()
	defer dataLock.RUnlock()
	return ServePage(c, "articles_home", fiber.Map{
		"AllArticles": allArticles,
		"PageTitle": "articles - gavin m's blog",
		"PageId": "home-articles",
	})
}

func ServeArticle(c *fiber.Ctx) error {
	dataLock.RLock()
	defer dataLock.RUnlock()
	article, ok := articlesMap[c.Params("article")]
	if !ok {
		return c.RedirectToRoute("/", fiber.Map{}, 404)
	}
	return ServePage(c, "article", fiber.Map{
		"Article": article,
		"PageTitle": article.Title,
		"PageId": "article",
	})
}

func ServeSeriesHome(c *fiber.Ctx) error {
	dataLock.RLock()
	defer dataLock.RUnlock()
	return ServePage(c, "series_home", fiber.Map{
		"AllSeries": allSeries,
		"PageTitle": "series - gavin m's blog",
		"PageId": "home-series",
	})
}

func ServeSeriesArticle(c *fiber.Ctx) error {
	dataLock.RLock()
	defer dataLock.RUnlock()
	series, ok := seriesMap[c.Params("series")]
	if !ok {
		return c.RedirectToRoute("/series", fiber.Map{}, 404)
	}
	article, ok := series.PartsMap[c.Params("article")]
	if !ok {
		return c.RedirectToRoute("/series#" + series.Path, fiber.Map{}, 404)
	}
	return ServePage(c, "article", fiber.Map{
		"Article": article,
		"PageTitle": article.Title + " - " + series.Title,
		"PageId": "article",
	})
}

type RawArticle struct {
	Title string `json:"title"`
	Subtitle string `json:"subtitle"`
	Datetime string `json:"datetime"`
	Filename string `json:"filename"`
}

type RawSeries struct {
	Title string `json:"title"`
	Description string `json:"description"`
	Dirname string `json:"dirname"`
	Parts []RawArticle `json:"parts"`
}

func PathFromTitle(title string) string {
	trimmed := []string{}
	for _, p := range strings.Split(title, "-") {
		trimmed = append(trimmed, strings.Trim(p, " -/"))
	}
	return strings.ToLower(
		strings.ReplaceAll(strings.Join(trimmed, "-"), " ", "-"))
}

func MdToArticle(md []byte) []byte {
	extensions := parser.CommonExtensions | parser.LaxHTMLBlocks | parser.HardLineBreak | parser.Footnotes | parser.NoEmptyLineBeforeBlock | parser.OrderedListStart | parser.Attributes | parser.EmptyLinesBreakList | parser.Includes
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	html_flags := mdHtml.CommonFlags | mdHtml.SkipHTML | mdHtml.NoreferrerLinks | mdHtml.TOC | mdHtml.FootnoteReturnLinks | mdHtml.LazyLoadImages
	renderer := mdHtml.NewRenderer(mdHtml.RendererOptions{Flags: html_flags})

	return markdown.Render(doc, renderer)
}

func (raw *RawArticle) ToArticle(series *Series) Article {
	layout := "2006-01-02 15:04"
	datetime, err := time.Parse(layout, raw.Datetime)
	if err != nil {
		datetime = time.Now()
	}
	path := PathFromTitle(raw.Title)
	var dirname string
	var url string
	if series != nil {
		dirname = "series/" + series.Dirname
		url = "/series/" + series.Path + "/" + path
	} else {
		dirname = "articles"
		url = "/articles/" + path
	}
	md_body, err := os.ReadFile("data/" + dirname + "/" + raw.Filename)
	if err != nil {
		log.Print("Failed to read data/" + dirname + "/" + raw.Filename + ": ")
		log.Print(err)
		md_body = []byte{}
	}
	return Article{
		Path: path,
		Url: url,
		Title: raw.Title,
		Subtitle: raw.Subtitle,
		Date: datetime.Format("01/02/06"),
		UnixMillis: datetime.UnixMilli(),
		HTMLBody: template.HTML(MdToArticle(md_body)),
		Series: series,
	}
}

func (raw *RawSeries) ToSeries() Series {
	layout := "2006-01-02 15:04"
	timestamp := int64(0)
	for _, part := range raw.Parts {
		part_dt, err := time.Parse(layout, part.Datetime)
		if err != nil {
			continue
		}
		timestamp = max(timestamp, part_dt.UnixMilli())
	}
	if timestamp == 0 {
		timestamp = time.Now().UnixMilli()
	}
	path := PathFromTitle(raw.Title)
	series := Series{
		Path: path,
		Dirname: raw.Dirname,
		Title: raw.Title,
		Description: raw.Description,
		Date: time.UnixMilli(timestamp).Format("01/02/06"),
		UnixMillis: timestamp,
	}
	return series
}

func RetrieveData() error {
	articles_content, err := os.ReadFile("data/articles.json")
	if err != nil {
		log.Println("Failed to read articles.json")
		return err
	}

	raw_articles := []RawArticle{}
	err = json.Unmarshal(articles_content, &raw_articles)
	if err != nil {
		log.Println("Failed to parse articles.json")
		return err
	}

	articles := map[string]Article{}
	for _, raw := range raw_articles {
		article := raw.ToArticle(nil)
		articles[article.Path] = article
	}

	series_content, err := os.ReadFile("data/series.json")
	if err != nil {
		log.Println("Failed to read series.json")
		return err
	}

	raw_series := []RawSeries{}
	err = json.Unmarshal(series_content, &raw_series)
	if err != nil {
		log.Println("Failed to parse series.json")
		return err
	}

	series := []Series{}
	series_ptrs := map[string]*Series{}
	for i, raw := range raw_series {
		s := raw.ToSeries()
		series = append(series, s)
		series_ptrs[s.Path] = &series[i]
		for _, a := range raw.Parts {
			article := a.ToArticle(&series[i])
			articles[s.Path + "/" + article.Path] = article
		}
	}

	dataLock.Lock()
	defer dataLock.Unlock()

	allArticles = slices.SortedStableFunc(maps.Values(articles), func(a1, a2 Article) int {
		return int(a2.UnixMillis - a1.UnixMillis)
	})

	articlesMap = map[string]*Article{}
	articles_series_map := map[*Series][]*Article{}
	for i := range allArticles {
		if allArticles[i].Series == nil {
			articlesMap[allArticles[i].Path] = &allArticles[i]
		} else {
			articles_series_map[allArticles[i].Series] =
				append(articles_series_map[allArticles[i].Series], &allArticles[i])
		}
	}

	allSeries = slices.SortedStableFunc(slices.Values(series), func(s1, s2 Series) int {
		return int(s2.UnixMillis - s1.UnixMillis)
	})
	seriesMap = map[string]*Series{}
	for i := range allSeries {
		seriesMap[allSeries[i].Path] = &allSeries[i]
		allSeries[i].PartsMap = map[string]*Article{}
		for _, article_ptr := range articles_series_map[series_ptrs[allSeries[i].Path]] {
			allSeries[i].Parts = append(allSeries[i].Parts, article_ptr)
			allSeries[i].PartsMap[article_ptr.Path] = article_ptr
			article_ptr.Series = &allSeries[i]
		}
		slices.SortStableFunc(allSeries[i].Parts, func(a1, a2 *Article) int {
			return int(a2.UnixMillis - a1.UnixMillis)
		})
	}

	return nil
}

func RetrieveDataFromSocket() {
	RetrieveData()

	os.Remove("data/site.sock")
	socket, err := net.Listen("unix", "data/site.sock")
	if err != nil {
		log.Fatal(err)
	}

	os.Chmod("data/site.sock", 0o777)
	if err != nil {
		log.Fatal(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		os.Remove("data/site.sock")
		os.Exit(1)
	}()

	for {
		conn, err := socket.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go func(conn net.Conn) {
			buf := make([]byte, 16)

			n, err := conn.Read(buf)
			if err != nil {
				log.Println(err)
				return
			}

			conn.Close()

			if string(buf[:n]) == "reload" {
				RetrieveData()
			}
		}(conn)
	}
}

func main() {
	go RetrieveDataFromSocket()
	
	app := fiber.New(fiber.Config{
		Views: html.NewFileSystem(http.FS(viewsDir), ".html"),
	})

	app.Use(helmet.New())
	app.Use(cache.New(cache.Config{
		KeyGenerator: func(c *fiber.Ctx) string {
			if htmx.IsHTMX(c) {
				return "H--" + strings.Clone(c.Path())
			} else {
				return "N--" + strings.Clone(c.Path())
			}
		},
		Expiration: 20 * time.Minute,
		CacheControl: true,
		StoreResponseHeaders: true,
	}))
	app.Use("/assets", filesystem.New(filesystem.Config{
		Root: http.FS(assetsDir),
		PathPrefix: "data/assets",
		Browse: true,
	}))
	app.Use(earlydata.New())

	app.Get("/", ServeHome)
	app.Get("/articles", ServeArticleHome)
	app.Get("/articles/:article", ServeArticle)
	app.Get("/series", ServeSeriesHome)
	app.Get("/series/:series/:article", ServeSeriesArticle)

	log.Fatal(app.Listen(":8081"))
}


