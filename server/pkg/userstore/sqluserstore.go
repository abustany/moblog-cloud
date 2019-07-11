package userstore

import (
	"bytes"
	"database/sql"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type SQLUserStore struct {
	db *sqlx.DB

	createUserStmt         *sqlx.NamedStmt
	updateUserDataStmt     *sql.Stmt
	updateUserPasswordStmt *sql.Stmt
	getUserStmt            *sqlx.Stmt
	authenticateUserStmt   *sql.Stmt
	deleteUserStmt         *sql.Stmt

	createBlogStmt *sql.Stmt
	updateBlogStmt *sql.Stmt
	getBlogStmt    *sqlx.Stmt
	listBlogsStmt  *sqlx.Stmt
	deleteBlogStmt *sql.Stmt
}

type userRecord struct {
	Username    string `db:"username"`
	DisplayName string `db:"displayname"`
	Salt        []byte `db:"salt"`
	Password    []byte `db:"password"`
}

func NewSQLUserStore(driverName string, dbUrl string) (*SQLUserStore, error) {
	db, err := sqlx.Connect(driverName, dbUrl)

	if err != nil {
		return nil, errors.Wrap(err, "Error while connecting to the database")
	}

	createUserStmt, err := db.PrepareNamed(`INSERT INTO users VALUES (:username, :displayname, :salt, :password)`)

	if err != nil {
		return nil, errors.Wrap(err, "Error while preparing create user statement")
	}

	updateUserDataStmt, err := db.Prepare(`UPDATE users SET displayname = $1 WHERE username = $2`)

	if err != nil {
		return nil, errors.Wrap(err, "Error while preparing update user data statement")
	}

	updateUserPasswordStmt, err := db.Prepare(`UPDATE users SET salt = $1, password = $2 WHERE username = $3`)

	if err != nil {
		return nil, errors.Wrap(err, "Error while preparing update user password statement")
	}

	getUserStmt, err := db.Preparex(`SELECT * FROM users WHERE username = $1`)

	if err != nil {
		return nil, errors.Wrap(err, "Error while preparing get user statement")
	}

	authenticateUserStmt, err := db.Prepare(`SELECT salt, password FROM users WHERE username = $1`)

	if err != nil {
		return nil, errors.Wrap(err, "Error while preparing authenticate user statement")
	}

	deleteUserStmt, err := db.Prepare(`DELETE FROM users WHERE username = $1`)

	if err != nil {
		return nil, errors.Wrap(err, "Error while preparing delete user statement")
	}

	createBlogStmt, err := db.Prepare(`INSERT INTO blogs VALUES ($1, $2, $3)`)

	if err != nil {
		return nil, errors.Wrap(err, "Error while preparing create blog statement")
	}

	updateBlogStmt, err := db.Prepare(`UPDATE blogs SET displayname = $1 WHERE username = $2 AND slug = $3`)

	if err != nil {
		return nil, errors.Wrap(err, "Error while preparing update blog statement")
	}

	getBlogStmt, err := db.Preparex(`SELECT slug, displayname FROM blogs WHERE username = $1 AND slug = $2`)

	if err != nil {
		return nil, errors.Wrap(err, "Error while preparing get blog statement")
	}

	listBlogsStmt, err := db.Preparex(`SELECT slug, displayname FROM blogs WHERE username = $1`)

	if err != nil {
		return nil, errors.Wrap(err, "Error while preparing list blogs statement")
	}

	deleteBlogStmt, err := db.Prepare(`DELETE FROM blogs WHERE username = $1 AND slug = $2`)

	if err != nil {
		return nil, errors.Wrap(err, "Error while preparing delete blog statement")
	}

	return &SQLUserStore{
		db,

		createUserStmt,
		updateUserDataStmt,
		updateUserPasswordStmt,
		getUserStmt,
		authenticateUserStmt,
		deleteUserStmt,

		createBlogStmt,
		updateBlogStmt,
		getBlogStmt,
		listBlogsStmt,
		deleteBlogStmt,
	}, nil
}

func (s *SQLUserStore) CreateUser(user User) error {
	if err := validateUser(user, false); err != nil {
		return err
	}

	salt, password, err := hashNewPassword(user.Password)

	if err != nil {
		return errors.Wrap(err, "Error while hashing password")
	}

	record := userRecord{
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Salt:        salt,
		Password:    password,
	}

	if _, err := s.createUserStmt.Exec(&record); err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return ErrAlreadyExists
		}

		return errors.Wrap(err, "Error while creating user")
	}

	return nil
}

func (s *SQLUserStore) inTx(do func(tx *sql.Tx) error) error {
	tx, err := s.db.Begin()

	if err != nil {
		return errors.Wrap(err, "Error while starting transaction")
	}

	if err := do(tx); err != nil {
		if rollbackError := tx.Rollback(); rollbackError != nil {
			return errors.Wrapf(rollbackError, "Error while rolling back transaction because of error %s", err.Error())
		}

		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "Error while committing transaction")
	}

	return nil
}

func (s *SQLUserStore) UpdateUser(user User) error {
	if err := validateUser(user, true); err != nil {
		return err
	}

	err := s.inTx(func(tx *sql.Tx) error {
		var result sql.Result
		var err error

		if result, err = tx.Stmt(s.updateUserDataStmt).Exec(user.DisplayName, user.Username); err != nil {
			return errors.Wrap(err, "Error while updating user data")
		}

		rowsAffected, err := result.RowsAffected()

		if err != nil {
			return errors.Wrap(err, "Error counting the affected rows")
		}

		if rowsAffected != 1 {
			return ErrDoesNotExist
		}

		if user.Password != "" {
			var salt []byte
			var oldHash []byte

			err := s.authenticateUserStmt.QueryRow(user.Username).Scan(&salt, &oldHash)

			if err == sql.ErrNoRows {
				return errors.New("No password row!?")
			}

			password := hashPassword(user.Password, salt)

			if _, err := tx.Stmt(s.updateUserPasswordStmt).Exec(salt, password, user.Username); err != nil {
				return errors.Wrap(err, "Error while updating user password")
			}
		}

		return nil
	})

	return errors.Wrap(err, "Error while updating user")
}

func (s *SQLUserStore) GetUser(username string) (*User, error) {
	var record userRecord
	err := s.getUserStmt.Get(&record, username)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "Error while fetching user")
	}

	return &User{
		Username:    record.Username,
		DisplayName: record.DisplayName,
	}, nil
}

func (s *SQLUserStore) AuthenticateUser(username, password string) (bool, error) {
	var salt []byte
	var dbHash []byte

	err := s.authenticateUserStmt.QueryRow(username).Scan(&salt, &dbHash)

	if err == sql.ErrNoRows {
		return false, nil
	}

	hash := hashPassword(password, salt)

	if err != nil {
		return false, errors.Wrap(err, "Error while hashing password")
	}

	if err != nil {
		return false, errors.Wrap(err, "Error while authenticating user")
	}

	return bytes.Equal(hash, dbHash), nil
}

func (s *SQLUserStore) DeleteUser(username string) error {
	res, err := s.deleteUserStmt.Exec(username)

	if err != nil {
		return errors.Wrap(err, "Error while deleting user")
	}

	rowsAffected, err := res.RowsAffected()

	if err != nil {
		return errors.Wrap(err, "Error while counting affected rows")
	}

	if rowsAffected != 1 {
		return ErrDoesNotExist
	}

	return nil
}

func (s *SQLUserStore) AddBlog(username string, blog Blog) error {
	if err := validateBlog(blog); err != nil {
		return err
	}

	if _, err := s.createBlogStmt.Exec(username, blog.Slug, blog.DisplayName); err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return ErrBlogAlreadyExists
		}

		return errors.Wrap(err, "Error while creating blog")
	}

	return nil
}

func (s *SQLUserStore) UpdateBlog(username string, blog Blog) error {
	if err := validateBlog(blog); err != nil {
		return err
	}

	result, err := s.updateBlogStmt.Exec(blog.DisplayName, username, blog.Slug)

	if err != nil {
		return errors.Wrap(err, "Error while updating blog")
	}

	rowsAffected, err := result.RowsAffected()

	if err != nil {
		return errors.Wrap(err, "Error counting the affected rows")
	}

	if rowsAffected != 1 {
		return ErrBlogDoesNotExist
	}

	return nil
}

func (s *SQLUserStore) GetBlog(username, blogSlug string) (*Blog, error) {
	var blog Blog
	err := s.getBlogStmt.Get(&blog, username, blogSlug)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "Error while fetching blog")
	}

	return &blog, nil
}

func (s *SQLUserStore) ListBlogs(username string) ([]Blog, error) {
	var blogs []Blog
	err := s.listBlogsStmt.Select(&blogs, username)

	if err != nil {
		return nil, errors.Wrap(err, "Error while fetching blogs")
	}

	return blogs, nil
}

func (s *SQLUserStore) DeleteBlog(username, blogSlug string) error {
	res, err := s.deleteBlogStmt.Exec(username, blogSlug)

	if err != nil {
		return errors.Wrap(err, "Error while deleting blog")
	}

	rowsAffected, err := res.RowsAffected()

	if err != nil {
		return errors.Wrap(err, "Error while counting affected rows")
	}

	if rowsAffected != 1 {
		return ErrBlogDoesNotExist
	}

	return nil
}
