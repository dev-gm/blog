package main

import (
	"encoding/json"
	"strconv"
	"time"
	"golang.org/x/crypto/bcrypt"
	"github.com/gofiber/fiber/v2"
	"github.com/sopa0/htmx-fiber"
)

func serveAdmin(c *fiber.Ctx, data fiber.Map, name string) error {
	if htmx.IsHTMX(c) {
		htmx.NewResponse().Retarget("#body-inner").Write(c)
		return c.Render(name, data)
	} else {
		return c.Render(name, data,
			"views/nested/body",
			"views/nested/admin-sidebar",
			"views/nested/index",
		)
	}
}

func serveAdminPostReq(c *fiber.Ctx, data fiber.Map, name string) error {
	if htmx.IsHTMX(c) {
		htmx.NewResponse().Retarget("#body-inner").Write(c)
		return c.Render(name, data)
	} else {
		return c.SendStatus(200)
	}
}

// USERS

// POST / - update user data - name or password - requires full auth
func UpdateUser(c *fiber.Ctx) error {
	payload := struct {
		FullName string `json:"full_name"`
		Password string `json:"password"`
	}{}
	err := json.Unmarshal(c.Body(), &payload)
	if err != nil {
		c.SendString("Failed to parse JSON")
		return c.SendStatus(400)
	}
	user, _ := c.Locals("user").(User)
	var password_hash []byte
	err = bcrypt.CompareHashAndPassword(password_hash, []byte(payload.Password))
	if err != nil {
		c.SendString("Failed to hash password")
		return c.SendStatus(400)
	}
	result := db.Where(&user).Updates(&User{
		FullName: payload.FullName,
		PasswordHash: password_hash,
	})
	if result.Error != nil {
		c.SendString("Failed to update user")
		return c.SendStatus(500)
	}
	result = db.First(&user, &User{})
	return serveAdminPostReq(c, fiber.Map{
		"PageTitle": "Admin - Home",
		"User": user,
		"PageType": "admin",
		"Page": "home",
	}, "views/admin/home")
}

// GET /
func ServeAdminHome(c *fiber.Ctx) error {
	user, _ := c.Locals("user").(User)
	return serveAdmin(c, fiber.Map{
		"PageTitle": "Admin - Home",
		"User": user,
		"PageType": "admin",
		"Page": "home",
	}, "views/admin/home")
}

// POSTS

// POST /posts - create new post - requires token
func CreatePost(c *fiber.Ctx) error {
	payload := struct {
		Title string `json:"title"`
		Subtitle string `json:"subtitle"`
		Body string `json:"body"`
	}{}
	err := json.Unmarshal(c.Body(), &payload)
	if err != nil {
		c.SendString("Failed to parse JSON")
		return c.SendStatus(400)
	}
	user, _ := c.Locals("user").(User)
	result := db.Create(&Post{
		Url: GetPostUrl(time.Now(), payload.Title),
		Title: payload.Title,
		Subtitle: payload.Subtitle,
		Body: payload.Body,
		UserName: user.UserName,
	})
	if result.Error != nil {
		c.SendString("Failed to create post")
		return c.SendStatus(500)
	}
	posts := []Post{}
	result = db.Find(&posts, &Post{Author: user})
	if result.Error != nil {
		c.SendString("Failed to get posts")
		return c.SendStatus(404)
	}
	fmt_posts := make([]PostFmtDate, len(posts))
	for i, post := range posts {
		fmt_posts[i] = post.ToPostFmtDate()
	}
	return serveAdminPostReq(c, fiber.Map{
		"PageTitle": "Admin - Posts",
		"Posts": fmt_posts,
		"PageType": "admin",
		"Page": "posts",
	}, "views/admin/posts")
}

// GET /posts/new
func ServeAdminNewPost(c *fiber.Ctx) error {
	return serveAdmin(c, fiber.Map{
		"PageTitle": "Admin - New Post",
		"PageType": "admin",
		"Page": "post-new",
	}, "views/admin/new_post")
}

// PUT /posts/:id - update post - requries full auth
func UpdatePost(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		c.SendString("Got invalid id")
		return c.SendStatus(400)
	}
	payload := struct {
		Title string `json:"title"`
		Subtitle string `json:"subtitle"`
		Body string `json:"body"`
	}{}
	err = json.Unmarshal(c.Body(), &payload)
	if err != nil {
		c.SendString("Failed to parse JSON")
		return c.SendStatus(400)
	}
	post := Post{
		Title: payload.Title,
		Subtitle: payload.Subtitle,
		Body: payload.Body,
	}
	user, _ := c.Locals("user").(User)
	result := db.Where(&Post{Id: id, Author: user}).Updates(&post)
	if result.Error != nil {
		c.SendString("Failed to update post")
		return c.SendStatus(500)
	}
	post = Post{}
	result = db.First(&post, &Post{
		Id: uint64(c.QueryFloat("id")),
		Author: user,
	})
	if result.Error != nil {
		c.SendString("Failed to get posts")
		return c.SendStatus(404)
	}
	fmt_post := post.ToPostFmtDate()
	return serveAdminPostReq(c, fiber.Map{
		"PageTitle": "Admin - " + post.Title,
		"Post": fmt_post,
		"PageType": "admin",
		"Page": "post",
	}, "views/admin/post")
}

// DELETE /posts/delete/:id - delete post - requires full auth
func DeletePost(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		c.SendString("Got invalid id")
		return c.SendStatus(400)
	}
	user, _ := c.Locals("user").(User)
	result := db.Delete(&Post{Id: id, Author: user})
	if result.Error != nil {
		c.SendString("Failed to delete post")
		return c.SendStatus(500)
	}
	posts := []Post{}
	result = db.Find(&posts, &Post{Author: user})
	if result.Error != nil {
		c.SendString("Failed to get posts")
		return c.SendStatus(404)
	}
	fmt_posts := make([]PostFmtDate, len(posts))
	for i, post := range posts {
		fmt_posts[i] = post.ToPostFmtDate()
	}
	return serveAdminPostReq(c, fiber.Map{
		"PageTitle": "Admin - Posts",
		"Posts": fmt_posts,
		"PageType": "admin",
		"Page": "post-delete",
	}, "views/admin/posts")
}

// GET /posts/delete/:id
func ServeAdminDeletePost(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		c.SendString("Got invalid id")
		return c.SendStatus(400)
	}
	return serveAdmin(c, fiber.Map{
		"PageTitle": "Admin - Posts",
		"PostId": id,
		"PageType": "admin",
		"Page": "post-delete",
	}, "views/admin/delete_post")
}

// GET /posts
func ServeAdminPosts(c *fiber.Ctx) error {
	user, _ := c.Locals("user").(User)
	posts := []Post{}
	result := db.Find(&posts, &Post{Author: user})
	if result.Error != nil {
		c.SendString("Failed to get posts")
		return c.SendStatus(404)
	}
	fmt_posts := make([]PostFmtDate, len(posts))
	for i, post := range posts {
		fmt_posts[i] = post.ToPostFmtDate()
	}
	return serveAdmin(c, fiber.Map{
		"PageTitle": "Admin - Posts",
		"Posts": fmt_posts,
		"PageType": "admin",
		"Page": "posts",
	}, "views/admin/posts")
}

// GET /posts/:id
func ServeAdminPost(c *fiber.Ctx) error {
	user, _ := c.Locals("user").(User)
	post := Post{}
	result := db.First(&post, &Post{
		Id: uint64(c.QueryFloat("id")),
		Author: user,
	})
	if result.Error != nil {
		c.SendString("Failed to get posts")
		return c.SendStatus(404)
	}
	fmt_post := post.ToPostFmtDate()
	return serveAdmin(c, fiber.Map{
		"PageTitle": "Admin - " + post.Title,
		"Post": fmt_post,
		"PageType": "admin",
		"Page": "post",
	}, "views/admin/post")
}

func Admin() *fiber.App {
	admin := fiber.New()

	admin.Use(PasetoMiddleware)

	admin.Get("/", ServeAdminHome)
	admin.Post("/", UpdateUser)

	admin.Get("/posts", ServeAdminPosts)
	
	admin.Get("/posts/new", ServeAdminNewPost)
	admin.Post("/posts", CreatePost)

	admin.Get("/posts/:id", ServeAdminPost)
	admin.Post("/posts/:id", UpdatePost)

	admin.Get("/posts/delete/:id", ServeAdminDeletePost)
	admin.Post("/posts/delete/:id", DeletePost)

	return admin
}
