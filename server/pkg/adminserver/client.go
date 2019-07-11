package adminserver

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"

	"github.com/pkg/errors"

	"github.com/abustany/moblog-cloud/pkg/rpcclient"
	"github.com/abustany/moblog-cloud/pkg/userstore"
)

type Client struct {
	url    string
	client *rpcclient.Client
}

func NewClient(url string) (*Client, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})

	if err != nil {
		return nil, errors.Wrap(err, "Error while creating cookie jar")
	}

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Jar:     jar,
	}

	return NewClientWithOptions(url, httpClient)
}

func NewClientWithOptions(url string, httpClient *http.Client) (*Client, error) {
	rpcClient := rpcclient.New(url)
	rpcClient.Client = httpClient

	return &Client{url, rpcClient}, nil
}

func (c *Client) AuthCookie() (*http.Cookie, error) {
	parsedUrl, err := url.Parse(c.url)

	if err != nil {
		return nil, errors.Wrap(err, "Error while parsing server url")
	}

	for _, cookie := range c.client.Client.Jar.Cookies(parsedUrl) {
		if cookie.Name == AuthCookieName {
			return cookie, nil
		}
	}

	return nil, nil
}

func (c *Client) Login(username, password string) error {
	values := url.Values{
		"username": []string{username},
		"password": []string{password},
	}

	res, err := c.client.Client.Post(c.url+"/login", "application/x-www-form-urlencoded", strings.NewReader(values.Encode()))

	if err != nil {
		return errors.Wrap(err, "Error while sending login request")
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return errors.Errorf("Invalid username or password (status code %d)", res.StatusCode)
	}

	return nil
}

func (c *Client) Logout() error {
	res, err := c.client.Client.Post(c.url+"/logout", "application/x-www-form-urlencoded", nil)

	if err != nil {
		return errors.Wrap(err, "Error while sending logout request")
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return errors.Wrapf(err, "Logout request returned status %d", res.StatusCode)
	}

	return nil
}

func (c *Client) CreateUser(user userstore.User) error {
	return c.client.Call("Users.Create", &user, &CreateUserReply{})
}

func (c *Client) UpdateUser(user userstore.User) error {
	return c.client.Call("Users.Update", &user, &UpdateUserReply{})
}

func (c *Client) GetUser(username string) (user userstore.User, err error) {
	err = c.client.Call("Users.Get", &GetUserArgs{username}, &user)
	return
}

func (c *Client) Whoami() (user userstore.User, err error) {
	err = c.client.Call("Users.Whoami", &WhoamiArgs{}, &user)
	return
}

func (c *Client) DeleteUser(username string) error {
	return c.client.Call("Users.Delete", &DeleteUserArgs{username}, &DeleteUserReply{})
}

func (c *Client) CreateBlog(blog userstore.Blog) error {
	return c.client.Call("Blogs.Create", &blog, &CreateBlogReply{})
}

func (c *Client) UpdateBlog(blog userstore.Blog) error {
	return c.client.Call("Blogs.Update", &blog, &UpdateBlogReply{})
}

func (c *Client) GetBlog(slug string) (user userstore.Blog, err error) {
	err = c.client.Call("Blogs.Get", &GetBlogArgs{slug}, &user)
	return
}

func (c *Client) ListBlogs() (blogs []userstore.Blog, err error) {
	err = c.client.Call("Blogs.List", &ListBlogsArgs{}, &blogs)
	return
}

func (c *Client) DeleteBlog(slug string) error {
	return c.client.Call("Blogs.Delete", &DeleteBlogArgs{slug}, &DeleteBlogReply{})
}
