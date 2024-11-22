package main

import (
	"embed"
	"log"
	"net/http"
	"strings"
	"time"
	"golang.org/x/crypto/bcrypt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/gofiber/helmet/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

//go:embed views/*
var viewsFs embed.FS

var engine *html.Engine

func main() {
	var err error
	db, err = gorm.Open(sqlite.Open("blog.db"), &gorm.Config{
		NowFunc: func() time.Time { return time.Now().Local() },
	})
	if err != nil {
		log.Fatal(err)
	}

	err = db.AutoMigrate(&Post{}, &User{})
	if err != nil {
		log.Fatal(err)
	}

	{
		password_hash, err := bcrypt.GenerateFromPassword(
			[]byte("password"), bcrypt.DefaultCost,
		)
		if err != nil {
			log.Fatal(err)
		}
		result := db.Where(&User{UserName: "gavin"}).FirstOrCreate(&User{
			UserName: "gavin",
			FullName: "Gavin M",
			PasswordHash: password_hash,
		})
		if result.Error != nil {
			log.Println(result.Error)
		}
		title := "My First Post's Title"
		result = db.Where(&Post{Title: title}).FirstOrCreate(&Post{
			Title: title,
			Url: GetPostUrl(time.Now(), title),
			Body: strings.Repeat(strings.Repeat("Body 1 ", 20) + "\n", 100),
			UserName: "gavin",
		})
		if result.Error != nil {
			log.Println(result.Error)
		}
		title = "My Second Post's Title"
		result = db.Where(&Post{Title: title}).FirstOrCreate(&Post{
			Title: title,
			Url: GetPostUrl(time.Now(), title),
			Subtitle: strings.Repeat("My Second Post's Subtitle ", 2),
			Body: strings.Repeat(strings.Repeat("Body 2 ", 20) + "\n", 100),
			UserName: "gavin",
		})
		if result.Error != nil {
			log.Println(result.Error)
		}
	}

	root := fiber.New(fiber.Config{
		Views: html.NewFileSystem(http.FS(viewsFs), ".html"),
	})

	root.Use(helmet.New())

	root.Get("/", ServePosts)
	root.Get("/:year/:month/:day/:title", ServePost)

	root.Get("/login", ServeLogin)
	root.Post("/login", Login)

	root.Mount("/admin", Admin())

	log.Fatal(root.Listen(":3000"))
}
