package adminserver

import (
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/net/publicsuffix"

	"github.com/pkg/errors"

	"github.com/abustany/moblog-cloud/pkg/rpcclient"
	"github.com/abustany/moblog-cloud/pkg/userstore"
)

type Client struct {
	url    string
	client *rpcclient.Client

	// Value of type http.Cookie. We keep this one cached manually because we
	// need the full cookie (not just the name and value as returned by the HTTP
	// client's cookie jar) to return in AuthCookie(), so that it can for example
	// be serialized to a netscape cookie file that curl can use.
	authCookie atomic.Value
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

	rpcClient := rpcclient.New(url)
	rpcClient.Client = httpClient

	c := &Client{
		url:    url,
		client: rpcClient,
	}

	c.resetCachedAuthCookie()

	return c, nil
}

func (c *Client) AuthCookie() (*http.Cookie, error) {
	authCookie := c.authCookie.Load().(http.Cookie)

	if authCookie.Name == "" {
		return nil, nil
	}

	return &authCookie, nil
}

func (c *Client) setCachedAuthCookie(cookie *http.Cookie) error {
	parsedURL, err := url.Parse(c.url)

	if err != nil {
		return errors.Wrap(err, "Error while parsing client URL")
	}

	cookieCopy := *cookie

	if cookieCopy.Domain == "" {
		// Strip the port (if any) from the domain
		if strings.Contains(parsedURL.Host, ":") {
			host, _, err := net.SplitHostPort(parsedURL.Host)

			if err != nil {
				return errors.Wrap(err, "Error while parsing URL host")
			}

			cookieCopy.Domain = host
		} else {
			cookieCopy.Domain = parsedURL.Host
		}
	}

	if cookieCopy.Path == "" {
		cookieCopy.Path = parsedURL.Path
	}

	c.authCookie.Store(cookieCopy)

	return nil
}

func (c *Client) resetCachedAuthCookie() {
	c.authCookie.Store(http.Cookie{})
}

func (c *Client) SetAuthCookie(cookie *http.Cookie) error {
	if cookie != nil && cookie.Name != AuthCookieName {
		return errors.New("Invalid cookie name")
	}

	parsedURL, err := url.Parse(c.url)

	if err != nil {
		return errors.Wrap(err, "Error while parsing server url")
	}

	if cookie == nil {
		resetCookie := ResetAuthCookie()
		c.client.Client.Jar.SetCookies(parsedURL, []*http.Cookie{&resetCookie})
		c.resetCachedAuthCookie()
		return nil
	}

	c.client.Client.Jar.SetCookies(parsedURL, []*http.Cookie{cookie})

	if err := c.setCachedAuthCookie(cookie); err != nil {
		return errors.Wrap(err, "Error while caching auth cookie")
	}

	return nil
}

func (c *Client) login(username, password string) error {
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
		c.resetCachedAuthCookie()
		return errors.Errorf("Invalid username or password (status code %d)", res.StatusCode)
	}

	for _, cookie := range res.Cookies() {
		if cookie.Name == AuthCookieName {
			if err := c.setCachedAuthCookie(cookie); err != nil {
				return errors.Wrap(err, "Error while caching auth cookie")
			}
		}
	}

	return nil
}

func (c *Client) Login(username, password string) error {
	return c.login(username, password)
}

func (c *Client) RefreshSession() error {
	return c.login("", "")
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

	c.resetCachedAuthCookie()

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
	err = c.client.Call("Blogs.Get", &GetBlogArgs{Slug: slug}, &user)
	return
}

func (c *Client) GetUserBlog(username, slug string) (user userstore.Blog, err error) {
	err = c.client.Call("Blogs.Get", &GetBlogArgs{Username: username, Slug: slug}, &user)
	return
}

func (c *Client) ListBlogs() (blogs []userstore.Blog, err error) {
	err = c.client.Call("Blogs.List", &ListBlogsArgs{}, &blogs)
	return
}

func (c *Client) DeleteBlog(slug string) error {
	return c.client.Call("Blogs.Delete", &DeleteBlogArgs{slug}, &DeleteBlogReply{})
}
