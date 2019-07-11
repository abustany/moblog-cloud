package userstore

import (
	"crypto/rand"

	"github.com/pkg/errors"
	"golang.org/x/crypto/argon2"
)

type User struct {
	Username    string
	Password    string
	DisplayName string
}

type Blog struct {
	Slug        string
	DisplayName string
}

type UserStore interface {
	CreateUser(user User) error
	UpdateUser(user User) error
	GetUser(username string) (*User, error)
	AuthenticateUser(username, password string) (bool, error)
	DeleteUser(username string) error

	AddBlog(username string, blog Blog) error
	UpdateBlog(username string, blog Blog) error
	GetBlog(username, blogSlug string) (*Blog, error)
	ListBlogs(username string) ([]Blog, error)
	DeleteBlog(username, blogSlug string) error
}

var ErrAlreadyExists = errors.New("User already exists")
var ErrDoesNotExist = errors.New("User does not exist")
var ErrUsernameEmpty = errors.New("Username cannot be empty")
var ErrPasswordEmpty = errors.New("Password cannot be empty")

var ErrBlogAlreadyExists = errors.New("A blog with this slug already exists")
var ErrBlogDoesNotExist = errors.New("Blog does not exist")
var ErrBlogSlugEmpty = errors.New("Blog slug cannot be empty")

func validateUser(user User, allowEmptyPassword bool) error {
	if user.Username == "" {
		return ErrUsernameEmpty
	}

	if !allowEmptyPassword && user.Password == "" {
		return ErrPasswordEmpty
	}

	return nil
}

func hashNewPassword(password string) (salt []byte, hash []byte, err error) {
	// Argon parameters
	const time = 3           // seconds
	const memory = 32 * 1024 // kB
	const threads = 4
	const keyLen = 32  // bytes
	const saltLen = 16 // bytes

	salt = make([]byte, saltLen)
	_, err = rand.Read(salt)

	if err != nil {
		return
	}

	hash = hashPassword(password, salt)

	return
}

func hashPassword(password string, salt []byte) []byte {
	// Argon parameters
	const time = 3           // seconds
	const memory = 32 * 1024 // kB
	const threads = 4
	const keyLen = 32  // bytes
	const saltLen = 16 // bytes

	return argon2.Key([]byte(password), salt, time, memory, threads, keyLen)
}

func validateBlog(blog Blog) error {
	if blog.Slug == "" {
		return ErrBlogSlugEmpty
	}

	return nil
}
