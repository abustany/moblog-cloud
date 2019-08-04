package adminserver

import (
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	rpcJson "github.com/gorilla/rpc/json"
	"github.com/gorilla/securecookie"
	"github.com/pkg/errors"

	"github.com/abustany/moblog-cloud/pkg/middlewares"
	"github.com/abustany/moblog-cloud/pkg/sessionstore"
	"github.com/abustany/moblog-cloud/pkg/userstore"
)

type usersService struct {
	store        userstore.UserStore
	sessionStore sessionstore.SessionStore
}

var errInvalidParameter = errors.New("Invalid parameters")
var errRequireAuthentication = errors.New("This method requires authentication")
var errGetUnknownUser = errors.New("User does not exist")
var errUpdateInvalidUser = errors.New("You cannot update this user")
var errDeleteInvalidUser = errors.New("You cannot update this user")

var errUnknownBlog = errors.New("No blog with this slug")

type CreateUserReply struct{}

func (s *usersService) Create(r *http.Request, user *userstore.User, reply *CreateUserReply) error {
	if user == nil {
		return errInvalidParameter
	}

	if err := s.store.CreateUser(*user); err != nil {
		log.Printf("Error while creating user %s: %s", user.Username, err)
		return err
	}

	log.Printf("Created user %s", user.Username)

	return nil
}

type UpdateUserReply struct{}

func (s *usersService) Update(r *http.Request, user *userstore.User, reply *UpdateUserReply) error {
	session := SessionFromContext(r.Context())

	if session == nil {
		return errRequireAuthentication
	}

	if user == nil {
		return errInvalidParameter
	}

	if session.Username != user.Username {
		return errUpdateInvalidUser
	}

	if err := s.store.UpdateUser(*user); err != nil {
		log.Printf("Error while updating user %s: %s", user.Username, err)
		return err
	}

	log.Printf("Updated user %s", user.Username)

	return nil
}

type GetUserArgs struct {
	Username string
}

func (s *usersService) Get(r *http.Request, args *GetUserArgs, reply *userstore.User) error {
	user, err := s.store.GetUser(args.Username)

	if err != nil {
		log.Printf("Error while retrieving user %s: %s", args.Username, err)
		return err
	}

	if user == nil {
		return errGetUnknownUser
	}

	*reply = *user

	return nil
}

type WhoamiArgs struct{}

func (s *usersService) Whoami(r *http.Request, args *WhoamiArgs, reply *userstore.User) error {
	session := SessionFromContext(r.Context())

	if session == nil {
		return errRequireAuthentication
	}

	return s.Get(r, &GetUserArgs{session.Username}, reply)
}

type DeleteUserArgs struct {
	Username string
}

type DeleteUserReply struct{}

func (s *usersService) Delete(r *http.Request, args *DeleteUserArgs, reply *DeleteUserReply) error {
	session := SessionFromContext(r.Context())

	if session == nil {
		return errRequireAuthentication
	}

	if session.Username != args.Username {
		return errDeleteInvalidUser
	}

	if err := s.store.DeleteUser(args.Username); err != nil {
		log.Printf("Error while deleting user %s: %s", args.Username, err)
		return err
	}

	if err := s.sessionStore.Delete(session.Sid); err != nil {
		log.Printf("Error while deleting session after deleting user %s: %s", args.Username, err)
		return err
	}

	log.Printf("Deleted user %s", args.Username)

	return nil
}

type blogsService struct {
	store userstore.UserStore
}

type CreateBlogReply struct{}

func (s *blogsService) Create(r *http.Request, blog *userstore.Blog, reply *CreateBlogReply) error {
	session := SessionFromContext(r.Context())

	if session == nil {
		return errRequireAuthentication
	}

	if blog == nil {
		return errInvalidParameter
	}

	if err := s.store.AddBlog(session.Username, *blog); err != nil {
		log.Printf("Error while adding blog %s for user %s: %s", blog.Slug, session.Username, err)
		return err
	}

	log.Printf("Added blog %s for user %s", blog.Slug, session.Username)

	return nil
}

type UpdateBlogReply struct{}

func (s *blogsService) Update(r *http.Request, blog *userstore.Blog, reply *UpdateBlogReply) error {
	session := SessionFromContext(r.Context())

	if session == nil {
		return errRequireAuthentication
	}

	if blog == nil {
		return errInvalidParameter
	}

	if err := s.store.UpdateBlog(session.Username, *blog); err != nil {
		log.Printf("Error while updating blog %s for user %s: %s", blog.Slug, session.Username, err)
		return err
	}

	log.Printf("Updated blog %s for user %s", blog.Slug, session.Username)

	return nil
}

type GetBlogArgs struct {
	Username string
	Slug     string
}

func (s *blogsService) Get(r *http.Request, args *GetBlogArgs, reply *userstore.Blog) error {
	var username string

	if args.Username == "" {
		session := SessionFromContext(r.Context())

		if session == nil {
			return errRequireAuthentication
		}

		username = session.Username
	} else {
		username = args.Username
	}

	blog, err := s.store.GetBlog(username, args.Slug)

	if err != nil {
		log.Printf("Error retrieving blog %s for user %s: %s", args.Slug, username, err)
		return err
	}

	if blog == nil {
		return errUnknownBlog
	}

	*reply = *blog

	return nil
}

type ListBlogsArgs struct{}

type ListBlogsReply []userstore.Blog

func (s *blogsService) List(r *http.Request, args *ListBlogsArgs, reply *ListBlogsReply) error {
	session := SessionFromContext(r.Context())

	if session == nil {
		return errRequireAuthentication
	}

	blogs, err := s.store.ListBlogs(session.Username)

	if err != nil {
		log.Printf("Error retrieving blogs for user %s: %s", session.Username, err)
		return err
	}

	*reply = blogs

	return nil
}

type DeleteBlogArgs struct {
	Slug string
}

type DeleteBlogReply struct{}

func (s *blogsService) Delete(r *http.Request, args *DeleteBlogArgs, reply *DeleteBlogReply) error {
	session := SessionFromContext(r.Context())

	if session == nil {
		return errRequireAuthentication
	}

	if err := s.store.DeleteBlog(session.Username, args.Slug); err != nil {
		log.Printf("Error deleting blog %s for user %s: %s", args.Slug, session.Username, err)
		return err
	}

	log.Printf("Deleted blog %s for user %s", args.Slug, session.Username)

	return nil
}

type Server struct {
	router       *mux.Router
	secureCookie *securecookie.SecureCookie
	userStore    userstore.UserStore
	sessionStore sessionstore.SessionStore
}

func New(basePath string, secureCookie *securecookie.SecureCookie, userStore userstore.UserStore, sessionStore sessionstore.SessionStore) (*Server, error) {
	s := Server{
		router:       mux.NewRouter(),
		secureCookie: secureCookie,
		userStore:    userStore,
		sessionStore: sessionStore,
	}

	rpcServer := rpc.NewServer()
	if err := rpcServer.RegisterService(&usersService{userStore, sessionStore}, "Users"); err != nil {
		return nil, errors.Wrap(err, "Error while registering users service")
	}

	if err := rpcServer.RegisterService(&blogsService{userStore}, "Blogs"); err != nil {
		return nil, errors.Wrap(err, "Error while registering blogs service")
	}

	rpcServer.RegisterCodec(rpcJson.NewCodec(), "application/json")

	router := s.router

	if basePath != "" {
		router = router.PathPrefix(basePath).Subrouter()
	}

	router.Methods("POST").Path("/login").Handler(middlewares.WithLogging(http.HandlerFunc(s.loginHandler)))
	router.Methods("POST").Path("/logout").Handler(middlewares.WithLogging(WithSession(secureCookie, sessionStore, true, http.HandlerFunc(s.logoutHandler))))

	router.Methods("POST").Handler(middlewares.WithLogging(WithSession(secureCookie, sessionStore, false, rpcServer)))

	s.router.
		PathPrefix("/").
		Handler(middlewares.WithLogging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, "Not found")
		})))

	return &s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "Bad form data")
		return
	}

	username := r.Form.Get("username")
	password := r.Form.Get("password")

	authenticated, err := s.userStore.AuthenticateUser(username, password)

	if err != nil {
		log.Printf("Error while authenticating user %s: %s", username, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !authenticated {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	sessionExpires := time.Now().Add(30 * 24 * time.Hour)

	session := sessionstore.Session{
		Sid:      sessionstore.GenerateSessionID(),
		Expires:  sessionExpires,
		Username: username,
	}

	if err := s.sessionStore.Set(session); err != nil {
		log.Printf("Error while saving session for user %s: %s", username, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("Created new session %s for user %s", session.Sid, username)

	authCookie, err := EncodeAuthCookie(s.secureCookie, AuthCookie{
		SessionID: session.Sid,
	})

	if err != nil {
		log.Printf("Error while encoding session cookie for user %s: %s", username, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	authCookie.Expires = sessionExpires

	http.SetCookie(w, &authCookie)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	session := SessionFromContext(r.Context())

	if err := s.sessionStore.Delete(session.Sid); err != nil {
		log.Printf("Error while deleting session %s for %s: %s", session.Sid, session.Username, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("User %s logged out, destroying session %s", session.Username, session.Sid)

	authCookie := ResetAuthCookie()
	http.SetCookie(w, &authCookie)
	w.WriteHeader(http.StatusOK)
}
