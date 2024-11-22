package main

import (
	"regexp"
	"strings"
	"time"
	"gorm.io/gorm"
)

type User struct {
	Id uint64 `gorm:"primaryKey;<-:create"`
	UserName string `gorm:"uniqueIndex;<-:create"`
	FullName string
	PasswordHash []byte
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
}

type Post struct {
	Id uint64 `gorm:"primaryKey;<-:create"`
	Url string `gorm:"uniqueIndex;<-:create"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
	Title string
	Subtitle string
	Body string
	UserName string `gorm:"index"`
	Author User `gorm:"foreignKey:UserName;references:UserName;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func GetPostUrl(created_at time.Time, title string) string {
	stripped_title := strings.ToLower(strings.TrimSpace( strings.ReplaceAll(
		regexp.MustCompile(`[^a-zA-Z0-9 ]+`).ReplaceAllString(title, ""), " ", "-")))
	url := created_at.UTC().Format("06/01/02/") + stripped_title
	return url
}

type PostFmtDate struct {
	Post Post
	CreatedAt string
	SubtitleExists bool
}

func (post Post) ToPostFmtDate() PostFmtDate {
	return PostFmtDate{
		Post: post,
		CreatedAt: post.CreatedAt.Format("January _2, 2006"),
		SubtitleExists: len(post.Subtitle) > 0,
	}
}


