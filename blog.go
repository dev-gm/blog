package main
import (
	"github.com/gofiber/fiber/v2"
	"github.com/sopa0/htmx-fiber"
)

func serveBlog(c *fiber.Ctx, data fiber.Map, name string) error {
	if htmx.IsHTMX(c) {
		htmx.NewResponse().Retarget("#body-inner").Write(c)
		return c.Render(name, data)
	} else {
		return c.Render(name, data,
			"views/nested/body",
			"views/nested/navbar",
			"views/nested/index",
		)
	}
}

func ServePosts(c *fiber.Ctx) error {
	var posts []Post
	result := db.Preload("Author").Find(&posts)
	if result.Error != nil {
		c.SendString("Failed to get posts")
		return c.SendStatus(404)
	}
	fmt_posts := make([]PostFmtDate, len(posts))
	for i, post := range posts {
		fmt_posts[i] = post.ToPostFmtDate()
	}
	return serveBlog(c, fiber.Map{
		"PageTitle": "Posts",
		"NavHome": "gavinm.us/blog",
		"Posts": fmt_posts,
		"PageType": "blog",
		"Page": "posts",
	}, "views/posts")
}

func ServePost(c *fiber.Ctx) error {
	url := c.Params("year") + "/" + c.Params("month") + "/" + c.Params("day") + "/" + c.Params("title")
	post := Post{}
	result := db.Where(&Post{Url: url}).Preload("Author").First(&post)
	if result.Error != nil {
		c.SendString("Failed to get post")
		return c.SendStatus(404)
	}
	fmt_post := post.ToPostFmtDate()
	return serveBlog(c, fiber.Map{
		"PageTitle": post.Title,
		"NavHome": "gavinm.us/blog",
		"Post": fmt_post,
		"PageType": "blog",
		"Page": "post",
	}, "views/posts")
}

