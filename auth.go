package main
import (
	"log"
	"time"
	"math/rand/v2"
	"golang.org/x/crypto/bcrypt"
	"github.com/gofiber/fiber/v2"
	"github.com/o1egl/paseto"
)

var pasetoKey []byte = GenRandomKey()

func GenRandomKey() []byte {
	key := make([]byte, 32)
	chacha8 := rand.NewChaCha8([32]byte([]byte("XW^uG@e_q#_,q!R5mx{~OJwom^AGwv-@")))
	_, err := chacha8.Read(key)
	if err != nil {
		log.Fatal("Failed to generate paseto key")
	}
	return key
}

func ValidateBasicAuth(username, password string) bool {
	user := User{}
	result := db.First(&user, User{UserName: username})
	if result.Error != nil {
		return false
	}
	err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password))
	if err != nil {
		return false
	}
	return true
}

func PasetoMiddleware(c *fiber.Ctx) error {
	jsonToken := paseto.JSONToken{}
	err := paseto.NewV2().Decrypt(c.Cookies("user_token"), pasetoKey, &jsonToken, nil)
	if err != nil {
		c.ClearCookie("user_token")
		return c.Redirect("/login", 401)
	}
	user := User{}
	result := db.First(&user, User{UserName: jsonToken.Subject})
	if result.Error != nil {
		c.ClearCookie("user_token")
		return c.Redirect("/login", 401)
	}
	c.Locals("user", user)
	return c.Next()
}

// POST /login - retrieve username, password - send back paseto token - requires full auth
func Login(c *fiber.Ctx) error {
	now := time.Now()
	exp := now.Add(48 * time.Hour)
	jsonToken := paseto.JSONToken{
		Subject: c.Params("username"),
		IssuedAt: now,
		Expiration: exp,
		NotBefore: now,
	}
	token, err := paseto.NewV2().Encrypt(pasetoKey, jsonToken, nil)
	if err != nil {
		c.SendString("Failed to encrypt token")
		return c.SendStatus(500)
	}
	c.Cookie(&fiber.Cookie{
		Name: "user_token",
		Value: token,
	})
	return c.SendStatus(200)
}

// GET /login
func ServeLogin(c *fiber.Ctx) error {
	return c.Render("views/login", fiber.Map{
		"PageTitle": "Login",
	}, "views/nested/body", "views/nested/index")
}


