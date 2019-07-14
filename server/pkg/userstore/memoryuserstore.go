package userstore

import (
	"sync"
)

type memoryRecord struct {
	user  User
	blogs map[string]Blog
}

type MemoryUserStore struct {
	sync.Mutex

	users map[string]memoryRecord
}

func NewMemoryUserStore() (*MemoryUserStore, error) {
	return &MemoryUserStore{
		users: map[string]memoryRecord{},
	}, nil
}

func (s *MemoryUserStore) CreateUser(user User) error {
	if err := validateUser(user, false); err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	if _, exists := s.users[user.Username]; exists {
		return ErrAlreadyExists
	}

	s.users[user.Username] = memoryRecord{user: user, blogs: map[string]Blog{}}

	return nil
}

func (s *MemoryUserStore) UpdateUser(user User) error {
	if err := validateUser(user, true); err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	record, exists := s.users[user.Username]

	if !exists {
		return ErrDoesNotExist
	}

	record.user = user
	s.users[user.Username] = record

	return nil
}

func (s *MemoryUserStore) GetUser(username string) (*User, error) {
	s.Lock()
	defer s.Unlock()

	record, exists := s.users[username]

	if exists {
		record.user.Password = ""
		return &record.user, nil
	} else {
		return nil, nil
	}
}

func (s *MemoryUserStore) AuthenticateUser(username, password string) (bool, error) {
	s.Lock()
	defer s.Unlock()

	record, exists := s.users[username]

	if exists {
		return record.user.Password == password, nil
	} else {
		return false, nil
	}
}

func (s *MemoryUserStore) DeleteUser(username string) error {
	s.Lock()
	defer s.Unlock()

	if _, exists := s.users[username]; !exists {
		return ErrDoesNotExist
	}

	delete(s.users, username)
	return nil
}

func (s *MemoryUserStore) AddBlog(username string, blog Blog) error {
	if err := validateBlog(blog); err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	record, exists := s.users[username]

	if !exists {
		return ErrDoesNotExist
	}

	if _, exists := record.blogs[blog.Slug]; exists {
		return ErrBlogAlreadyExists
	}

	record.blogs[blog.Slug] = blog
	s.users[username] = record

	return nil
}

func (s *MemoryUserStore) UpdateBlog(username string, blog Blog) error {
	if err := validateBlog(blog); err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	record, exists := s.users[username]

	if !exists {
		return ErrDoesNotExist
	}

	if _, exists := record.blogs[blog.Slug]; !exists {
		return ErrBlogDoesNotExist
	}

	record.blogs[blog.Slug] = blog
	s.users[username] = record

	return nil
}

func (s *MemoryUserStore) GetBlog(username, blogSlug string) (*Blog, error) {
	s.Lock()
	defer s.Unlock()

	record, exists := s.users[username]

	if !exists {
		return nil, ErrDoesNotExist
	}

	if blog, exists := record.blogs[blogSlug]; exists {
		return &blog, nil
	}

	return nil, nil
}

func (s *MemoryUserStore) ListBlogs(username string) ([]Blog, error) {
	s.Lock()
	defer s.Unlock()

	record, exists := s.users[username]

	if !exists {
		return nil, ErrDoesNotExist
	}

	blogs := make([]Blog, 0, len(record.blogs))

	for _, blog := range record.blogs {
		blogs = append(blogs, blog)
	}

	return blogs, nil
}

func (s *MemoryUserStore) DeleteBlog(username, blogSlug string) error {
	s.Lock()
	defer s.Unlock()

	record, exists := s.users[username]

	if !exists {
		return ErrDoesNotExist
	}

	if _, exists := record.blogs[blogSlug]; !exists {
		return ErrBlogDoesNotExist
	}

	delete(record.blogs, blogSlug)

	return nil
}
