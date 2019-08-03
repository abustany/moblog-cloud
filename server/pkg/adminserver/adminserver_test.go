package adminserver_test

import (
	"net/http/httptest"
	"sort"
	"testing"

	_ "github.com/lib/pq"

	"github.com/abustany/moblog-cloud/pkg/adminserver"
	"github.com/abustany/moblog-cloud/pkg/testutils"
	"github.com/abustany/moblog-cloud/pkg/userstore"
)

type LoginFunc func(username, password string) error
type LogoutFunc func() error
type RPCFunc func(method string, param interface{}, reply interface{}) error

func TestUserService(t *testing.T) {
	testutils.FlushDB(t)

	server := httptest.NewServer(testutils.NewAdminServer(t))
	defer server.Close()

	client, err := adminserver.NewClient(server.URL)

	if err != nil {
		t.Fatalf("Error while creating RPC client: %s", err)
	}

	withClient := func(f func(*testing.T, *adminserver.Client)) func(*testing.T) {
		return func(t *testing.T) {
			f(t, client)
		}
	}

	t.Run("Create a new user", withClient(testCreateUser))
	t.Run("Get a user", withClient(testGetUser))
	t.Run("Update a user", withClient(testUpdateUser))
	t.Run("Update a different user", withClient(testUpdateDifferentUser))
	t.Run("Delete a user", withClient(testDeleteUser))
	t.Run("Log out", withClient(testLogout))

	t.Run("Create a blog", withClient(testCreateBlog))
	t.Run("Get a blog", withClient(testGetBlog))
	t.Run("Update a blog", withClient(testUpdateBlog))
	t.Run("Delete a blog", withClient(testDeleteBlog))
	t.Run("List blogs", withClient(testListBlogs))
}

func testCreateUser(t *testing.T, c *adminserver.Client) {
	user := userstore.User{
		Username:    "hello",
		Password:    "world",
		DisplayName: "Mr. Hello",
	}

	if err := c.CreateUser(user); err != nil {
		t.Errorf("Users.Create returned an error: %s", err)
	}

	if err := c.CreateUser(user); err == nil {
		t.Errorf("Expected error when creating a user with an existing name")
	}
}

func verifyUser(t *testing.T, user userstore.User, username, password, displayName string) {
	if user.Username != username {
		t.Errorf("Invalid username")
	}

	if user.Password != password {
		t.Errorf("Password should be empty")
	}

	if user.DisplayName != displayName {
		t.Errorf("Invalid display name")
	}
}

func testGetUser(t *testing.T, c *adminserver.Client) {
	if _, err := c.GetUser("nobody"); err == nil {
		t.Errorf("Expected an error when retrieving non existing user")
	}

	if user, err := c.GetUser("hello"); err != nil {
		t.Errorf("Error while retrieving user: %s", err)
	} else {
		verifyUser(t, user, "hello", "", "Mr. Hello")
	}
}

func testUpdateUser(t *testing.T, c *adminserver.Client) {
	user := userstore.User{
		Username:    "hello",
		DisplayName: "Sir Hello",
	}

	if err := c.UpdateUser(user); err == nil {
		t.Errorf("Expected error when updating a user without authentication")
	}

	if err := c.Login("hello", "world"); err != nil {
		t.Fatalf("Login failed: %s", err)
	}

	if err := c.UpdateUser(user); err != nil {
		t.Errorf("Users.Update returned an error: %s", err)
	}

	if updatedUser, err := c.GetUser(user.Username); err != nil {
		t.Errorf("Users.Get returned an error: %s", err)
	} else {
		verifyUser(t, updatedUser, "hello", "", "Sir Hello")
	}

	user.Password = "mundo"

	if err := c.UpdateUser(user); err != nil {
		t.Errorf("Users.Update return an error: %s", err)
	} else {
		if err := c.Login("hello", "world"); err == nil {
			t.Errorf("Login with old credentials should return an error")
		}

		if err := c.Login("hello", "mundo"); err != nil {
			t.Errorf("Login with new credentials return an error: %s", err)
		}
	}
}

func testUpdateDifferentUser(t *testing.T, c *adminserver.Client) {
	newUser := userstore.User{
		Username: "john",
		Password: "foo",
	}

	if err := c.CreateUser(newUser); err != nil {
		t.Errorf("Users.Create returned an error: %s", err)
	}

	// We're still logged in as hello from the previous test
	if err := c.UpdateUser(newUser); err == nil {
		t.Errorf("Expected error when updating a different user")
	}
}

func testDeleteUser(t *testing.T, c *adminserver.Client) {
	if err := c.DeleteUser("nobody"); err == nil {
		t.Errorf("Expected error when deleting non existing user")
	}

	if err := c.DeleteUser("hello"); err != nil {
		t.Errorf("Users.Delete returned an error: %s", err)
	}

	if err := c.DeleteUser("hello"); err == nil {
		t.Errorf("Expected error when deleting a deleted user")
	}

	if err := c.DeleteUser("john"); err == nil {
		t.Errorf("Expected error when deleting another user")
	}
}

func testLogout(t *testing.T, c *adminserver.Client) {
	john := userstore.User{
		Username: "john",
		Password: "foo",
	}

	if err := c.Login(john.Username, john.Password); err != nil {
		t.Errorf("Login as john failed: %s", err)
	}

	if err := c.UpdateUser(john); err != nil {
		t.Errorf("Users.Update returned an error")
	}

	if err := c.Logout(); err != nil {
		t.Errorf("Logout failed: %s", err)
	}

	if err := c.UpdateUser(john); err == nil {
		t.Errorf("Expected an error when calling Users.Update after logout")
	}
}

func testCreateBlog(t *testing.T, c *adminserver.Client) {
	blog := userstore.Blog{
		Slug:        "blog",
		DisplayName: "My fancy blog",
	}

	if err := c.CreateBlog(blog); err == nil {
		t.Errorf("Expected an error when calling Blogs.Create logged out")
	}

	if err := c.Login("john", "foo"); err != nil {
		t.Errorf("Error while logging in as john: %s", err)
	}

	if err := c.CreateBlog(userstore.Blog{}); err == nil {
		t.Errorf("Expected an error when creating a blog with an empty slug")
	}

	if err := c.CreateBlog(blog); err != nil {
		t.Errorf("Blogs.Create returned an error: %s", err)
	}

	if err := c.CreateBlog(blog); err == nil {
		t.Errorf("Expected an error when creating a blog with an existing slug")
	}
}

func verifyBlog(t *testing.T, blog userstore.Blog, slug, displayName string) {
	if blog.Slug != slug {
		t.Errorf("Invalid blog slug, expected %s, got %s", slug, blog.Slug)
	}

	if blog.DisplayName != displayName {
		t.Errorf("Invalid blog display name, expected %s, got %s", displayName, blog.DisplayName)
	}
}

func testGetBlog(t *testing.T, c *adminserver.Client) {
	if _, err := c.GetBlog("nothinghere"); err == nil {
		t.Errorf("Expected an error when retrieving a non existing blog")
	}

	if blog, err := c.GetBlog("blog"); err != nil {
		t.Errorf("Error while retrieving blog: %s", err)
	} else {
		verifyBlog(t, blog, "blog", "My fancy blog")
	}
}

func testUpdateBlog(t *testing.T, c *adminserver.Client) {
	blog := userstore.Blog{
		Slug:        "blog",
		DisplayName: "A more modern name",
	}

	if err := c.UpdateBlog(userstore.Blog{Slug: "nothinghere"}); err == nil {
		t.Errorf("Expected error when updating a non existing blog")
	}

	if err := c.UpdateBlog(blog); err != nil {
		t.Errorf("Blogs.Update returned an error: %s", err)
	}

	if updatedBlog, err := c.GetBlog(blog.Slug); err != nil {
		t.Errorf("Blogs.Get returned an error: %s", err)
	} else {
		verifyBlog(t, updatedBlog, blog.Slug, blog.DisplayName)
	}
}

func testDeleteBlog(t *testing.T, c *adminserver.Client) {
	if err := c.DeleteBlog("nothinghere"); err == nil {
		t.Errorf("Expected an error when deleting a non existing blog")
	}

	if err := c.DeleteBlog("blog"); err != nil {
		t.Errorf("Blogs.Delete returned an error: %s", err)
	}

	if err := c.DeleteBlog("blog"); err == nil {
		t.Errorf("Expected an error when deleting a deleted blog")
	}
}

type SortBySlug []userstore.Blog

func (l SortBySlug) Len() int {
	return len(l)
}

func (l SortBySlug) Swap(i, j int) {
	l[j], l[i] = l[i], l[j]
}

func (l SortBySlug) Less(i, j int) bool {
	return l[i].Slug < l[j].Slug
}

func testListBlogs(t *testing.T, c *adminserver.Client) {
	if blogs, err := c.ListBlogs(); err != nil {
		t.Errorf("Blogs.List returned an error: %s", err)
	} else {
		if len(blogs) != 0 {
			t.Errorf("Blogs.List returned %d entries, expected 0", len(blogs))
		}
	}

	blogs := []userstore.Blog{{Slug: "blog1"}, {Slug: "blog2"}}

	for i, blog := range blogs {
		if err := c.CreateBlog(blog); err != nil {
			t.Fatalf("Error while creating blog %d: %d", i, err)
		}

		if listedBlogs, err := c.ListBlogs(); err != nil {
			t.Errorf("Blogs.List returned an error after creating %d blogs", 1+i)
		} else {
			sort.Sort(SortBySlug(listedBlogs))

			for j := 0; j <= i; j++ {
				verifyBlog(t, listedBlogs[j], blogs[j].Slug, blogs[j].DisplayName)
			}
		}
	}
}
