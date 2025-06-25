package server

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	"github.com/mrdoob/glsl-sandbox/server/store"
	"golang.org/x/crypto/acme/autocert"
)

const (
	pathGallery = "./server/assets/gallery.html"
	pathThumbs  = "thumbs"
	pathCerts   = "certs"
	perPage     = 50

	// max session age 30 days
	sessionMaxAge = 86400 * 30
	sessionName   = "glslsandbox"
)

var ErrInvalidData = fmt.Errorf("invalid data")

type Template struct {
	templates *template.Template
}

func prepareTemplate() (*template.Template, error) {
	tpl := template.New("")
	tpl = tpl.Funcs(template.FuncMap{
		"checkboxID": func(id int) string {
			return fmt.Sprintf("hidden_%d", id)
		},
		"checked": func(b bool) string {
			if b {
				return "checked"
			}
			return ""
		},
	})
	tpl, err := tpl.ParseFiles(pathGallery)
	if err != nil {
		fmt.Println("template error", err.Error())
		return nil, err
	}

	return tpl, nil
}

func (t *Template) Render(
	w io.Writer, name string, data any, c echo.Context,
) error {
	tpl := t.templates
	if tpl == nil {
		var err error
		tpl, err = prepareTemplate()
		if err != nil {
			return err
		}
	}
	return tpl.ExecuteTemplate(w, name, data)
}

type Server struct {
	addr     string
	tlsAddr  string
	domains  []string
	echo     *echo.Echo
	template *Template
	effects  *store.Effects
	auth     *Auth
	dataPath string
	readOnly bool

	sessionStore sessions.Store
	callbackHost string
}

func New(
	addr string,
	tlsAddr string,
	domains string,
	e *store.Effects,
	auth *Auth,
	dataPath string,
	dev bool,
	readOnly bool,
	callbackServer string,
) (*Server, error) {
	var tpl *template.Template
	if !dev {
		var err error
		tpl, err = prepareTemplate()
		if err != nil {
			return nil, err
		}
	}

	if tlsAddr != "" && domains == "" {
		return nil, fmt.Errorf("cannot specify TLS_ADDR without DOMAINS")
	}

	return &Server{
		addr:    addr,
		tlsAddr: tlsAddr,
		domains: strings.Split(domains, ","),
		echo:    echo.New(),
		template: &Template{
			templates: tpl,
		},
		effects:      e,
		auth:         auth,
		dataPath:     dataPath,
		readOnly:     readOnly,
		callbackHost: callbackServer,
	}, nil
}

func (s *Server) Start() error {
	err := s.setup()
	if err != nil {
		return err
	}

	if s.tlsAddr != "" {
		go func() {
			err := s.echo.Start(s.addr)
			s.echo.Logger.Errorf("could not create http server: %s", err.Error())
		}()

		return s.echo.StartAutoTLS(s.tlsAddr)
	}

	return s.echo.Start(s.addr)
}

func (s *Server) setup() error {
	if s.tlsAddr != "" {
		certs := filepath.Join(s.dataPath, pathCerts)
		err := os.MkdirAll(certs, 0750)
		if err != nil {
			return fmt.Errorf("could not create certs directory: %w", err)
		}

		s.echo.AutoTLSManager.Cache = autocert.DirCache(certs)
		s.echo.AutoTLSManager.HostPolicy = autocert.HostWhitelist(s.domains...)

		s.echo.Pre(middleware.HTTPSRedirect())
	}

	err := s.setupAuth()
	if err != nil {
		return err
	}

	s.echo.Use(middleware.Recover())
	s.echo.Renderer = s.template
	s.echo.Logger.SetLevel(log.DEBUG)
	s.echo.Use(middleware.Logger())
	s.echo.Use(s.auth.Middleware(func(ctx echo.Context, err error) error {
		ctx.Logger().Debugf("middleware error: %w", err)
		return nil
	}))
	s.routes()

	return nil
}

func (s *Server) setupAuth() error {
	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		return errors.New("SESSION_SECRET must be set")
	}

	sessionStore := sessions.NewCookieStore([]byte("secret"))
	sessionStore.MaxAge(sessionMaxAge)
	gothic.Store = sessionStore

	s.sessionStore = sessionStore

	s.echo.Use(session.Middleware(sessionStore))

	githubClientID := os.Getenv("GITHUB_CLIENT_ID")
	githubSecret := os.Getenv("GITHUB_SECRET")

	if githubClientID == "" || githubSecret == "" {
		return errors.New("GITHUB_CLIENT_ID and GITHUB_SECRET must be set")
	}

	goth.UseProviders(
		github.New(githubClientID, githubSecret, s.callbackURL("github")),
	)

	return nil
}

func (s *Server) routes() {
	s.echo.GET("/", s.indexHandler)
	s.echo.GET("/e", s.effectHandler)
	s.echo.GET("/e_", s.effectHandler_)

	if !s.readOnly {
		s.echo.POST("/e", s.saveHandler)
	}

	cors := middleware.CORSWithConfig(middleware.CORSConfig{
		Skipper:      middleware.DefaultSkipper,
		AllowOrigins: []string{"*"},
	})
	s.echo.GET("/item/:id", s.itemHandler, cors)

	s.echo.Static("/thumbs", filepath.Join(s.dataPath, pathThumbs))
	s.echo.Static("/css", "./server/assets/css")
	s.echo.Static("/img", "./server/assets/img")
	s.echo.Static("/js", "./server/assets/js")
	s.echo.File("/diff", "./server/assets/diff.html")

	s.echo.File("/login", "./server/assets/login.html")
	s.echo.POST("/login", s.loginHandler)

	admin := s.echo.Group("/admin")
	admin.Use(s.auth.Middleware(func(c echo.Context, err error) error {
		c.Logger().Errorf("not authorized: %s", err.Error())
		return c.Redirect(http.StatusSeeOther, "/login")
	}))

	admin.GET("", s.adminHandler)
	admin.POST("", s.adminPostHandler)

	s.authRoutes()
}

func (s *Server) indexHandler(c echo.Context) error {
	sess, err := s.sessionStore.Get(c.Request(), sessionName)
	if err == nil {
		spew.Dump(sess.Values)
	} else {
		s.echo.Logger.Error(err)
	}

	return s.indexRender(c, false)
}

func (s *Server) adminHandler(c echo.Context) error {
	return s.indexRender(c, true)
}

// galleryEffect has information about each effect displayed in the gallery.
type galleryEffect struct {
	// ID is the effect identifier.
	ID int
	// Version is the latest effect version.
	Version int
	// Image holds the thumbnail name.
	Image string
	// Hidden tells if the effect has been moderated.
	Hidden bool
}

// galleryData has information about the current gallery page.
type galleryData struct {
	// Effects is an array with all the effects for the page.
	Effects []galleryEffect
	// URL is the path of the gallery. Can be "/" or "/admin".
	URL string
	// Page holds the current page number.
	Page int
	// IsPrevious is true if there is a previous page.
	IsPrevious bool
	// PreviousPage is the previous page number.
	PreviousPage string
	// IsNext is true if there is a next page.
	IsNext bool
	// NextPage is the next page number.
	NextPage string
	// Admin is true when accessing "/admin" path.
	Admin bool
	// ReadOnly tells the server is in read only mode.
	ReadOnly bool
	// LoggedIn tells if the user is logged in.
	LoggedIn bool
}

func (s *Server) indexRender(c echo.Context, admin bool) error {
	pString := c.QueryParam("page")
	if pString == "" {
		pString = "0"
	}
	page, err := strconv.Atoi(pString)
	if err != nil {
		page = 0
	}

	parent := -1
	if c.QueryParam("parent") != "" {
		parent, err = strconv.Atoi(c.QueryParam("parent"))
		if err != nil {
			parent = -1
		}
	}

	var p []store.Effect
	if parent > 0 {
		p, err = s.effects.PageSiblings(page, perPage, parent)
	} else {
		p, err = s.effects.Page(page, perPage, admin)
	}
	if err != nil {
		return c.String(http.StatusInternalServerError, "error")
	}

	effects := make([]galleryEffect, len(p))
	for i, e := range p {
		effects[i] = galleryEffect{
			ID:      e.ID,
			Version: len(e.Versions) - 1,
			Image:   path.Join("/thumbs", e.ImageName()),
			Hidden:  e.Hidden,
		}
	}

	url := "/"
	if admin {
		url = "/admin"
	}

	nextPage := fmt.Sprintf("%s?page=%d", url, page+1)
	previousPage := fmt.Sprintf("%s?page=%d", url, page-1)

	if parent > 0 {
		nextPage = fmt.Sprintf("%s&parent=%d", nextPage, parent)
		previousPage = fmt.Sprintf("%s&parent=%d", previousPage, parent)
	}

	loggedIn := true
	_, err = s.auth.GetUser(c)
	if err != nil {
		loggedIn = false
		s.echo.Logger.Debugf("user is not logged in", err)
	}

	d := galleryData{
		Effects:      effects,
		URL:          url,
		Page:         page,
		IsNext:       len(effects) == perPage,
		NextPage:     nextPage,
		IsPrevious:   page > 0,
		PreviousPage: previousPage,
		Admin:        admin,
		ReadOnly:     s.readOnly,
		LoggedIn:     loggedIn,
	}

	return c.Render(http.StatusOK, "gallery", d)
}

func (s *Server) effectHandler(c echo.Context) error {
	return c.File("./server/assets/index.html")
}

func (s *Server) effectHandler_(c echo.Context) error {
	return c.File("./static/index_.html")
}

type itemResponse struct {
	Code   string `json:"code"`
	User   string `json:"user"`
	Parent string `json:"parent,omitempty"`
}

func (s *Server) itemHandler(c echo.Context) error {
	param := c.Param("id")
	id, version, err := idVersion(param)
	if err != nil {
		return c.String(http.StatusBadRequest, "{}")
	}

	effect, err := s.effects.Effect(id)
	if err != nil {
		return c.String(http.StatusBadRequest, "{}")
	}

	if version >= len(effect.Versions) || version < 0 {
		return c.String(http.StatusNotFound, "{}")
	}

	parent := ""
	if effect.Parent > 0 {
		parent = fmt.Sprintf("/e#%d.%d", effect.Parent, effect.ParentVersion)
	}

	item := itemResponse{
		Code:   effect.Versions[version].Code,
		User:   effect.User,
		Parent: parent,
	}

	data, err := json.Marshal(item)
	if err != nil {
		return c.String(http.StatusInternalServerError, "{}")
	}

	return c.Blob(http.StatusOK, "application/json", data)
}

type saveQuery struct {
	Code   string `json:"code"`
	Image  string `json:"image"`
	User   string `json:"user"`
	CodeID string `json:"code_id"`
	Parent string `json:"parent"`
}

func (s *Server) saveHandler(c echo.Context) error {
	user, err := s.auth.CheckPermissions(c, store.RoleUser)
	if err != nil {
		c.Logger().Errorf("no permissions to save: %s", err.Error())
		return c.String(http.StatusBadRequest, "")
	}

	if c.Request().Body == nil {
		c.Logger().Errorf("empty body")
		return c.String(http.StatusBadRequest, "")
	}

	data, err := io.ReadAll(c.Request().Body)
	if err != nil {
		c.Logger().Errorf("could not read body: %s", err.Error())
		return c.String(http.StatusInternalServerError, "")
	}

	var save saveQuery
	err = json.Unmarshal(data, &save)
	if err != nil {
		c.Logger().Errorf("could not parse json: %s", err.Error())
		return c.String(http.StatusBadRequest, "")
	}

	parts := strings.Split(save.Image, ",")
	if len(parts) != 2 {
		c.Logger().Errorf("malformed encoded image")
		return c.String(http.StatusBadRequest, "")
	}
	imgData := parts[1]

	img, err := base64.StdEncoding.DecodeString(imgData)
	if err != nil {
		c.Logger().Errorf("could not decode image: %s", err.Error())
		return c.String(http.StatusBadRequest, "")
	}

	var id, version int
	var effect store.Effect

	// get current effect

	if save.CodeID != "" {
		parts = strings.Split(save.CodeID, ".")
		if len(parts) < 1 {
			c.Logger().Errorf("malformed code id: %s", save.CodeID)
			return c.String(http.StatusBadRequest, "")
		}

		id, err = strconv.Atoi(parts[0])
		if err != nil {
			c.Logger().Errorf("malformed code id: %s", err.Error())
			return c.String(http.StatusBadRequest, "")
		}

		if id > 0 {
			effect, err = s.effects.Effect(id)
			if err != nil {
				c.Logger().Errorf("could not get effect: %s", err.Error())
				return c.String(http.StatusBadRequest, "")
			}
		}
	}

	if save.CodeID == "" || user.ID != effect.ID {
		parent, parentVersion, err := idVersion(save.Parent)
		if err != nil {
			parent, parentVersion = -1, -1
		}

		id, err = s.effects.Add(parent, parentVersion, user.ID, save.Code)
		if err != nil {
			c.Logger().Errorf("could not save new effect: %s", err.Error())
			return c.String(http.StatusInternalServerError, "")
		}
	} else {
		version, err = s.effects.AddVersion(id, save.Code)
		if err != nil {
			c.Logger().Errorf("could not save new version: %s", err.Error())
			return c.String(http.StatusInternalServerError, "")
		}
	}

	err = saveImage(thumbPath(s.dataPath, id), img)
	if err != nil {
		c.Logger().Errorf("could not save image: %s", err.Error())
		return c.String(http.StatusInternalServerError, "")
	}

	answer := fmt.Sprintf("%d.%d", id, version)
	return c.String(http.StatusOK, answer)
}

func (s *Server) adminPostHandler(c echo.Context) error {
	pageTxt := c.FormValue("page")
	// TODO(jfontan): check error?
	page, _ := strconv.Atoi(pageTxt)
	url := fmt.Sprintf("/admin?page=%d", page)

	values, err := c.FormParams()
	if err != nil {
		c.Logger().Errorf("malformed form: %s", err.Error())
		return c.Redirect(http.StatusSeeOther, url)

	}

	on := make(map[int]struct{})
	for n, v := range values {
		if !strings.HasPrefix(n, "hidden_") {
			continue
		}
		if len(v) != 1 || v[0] != "on" {
			continue
		}

		parts := strings.Split(n, "_")
		if len(parts) != 2 {
			continue
		}

		id, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		on[id] = struct{}{}

		err = s.effects.Hide(id, true)
		if err != nil {
			c.Logger().Errorf("could not hide effect: %s", err.Error())
		}
	}

	e := values["effects"]
	for _, d := range e {
		id, err := strconv.Atoi(d)
		if err != nil {
			continue
		}

		if _, ok := on[id]; ok {
			continue
		}

		err = s.effects.Hide(id, false)
		if err != nil {
			c.Logger().Errorf("could not reveal effect: %s", err.Error())
		}
	}

	return c.Redirect(http.StatusSeeOther, url)
}

type loginData struct {
	Name     string `form:"name"`
	Password string `form:"password"`
}

// TODO(jfontan): disable user password login?
func (s *Server) loginHandler(c echo.Context) error {
	log := c.Logger()

	var l loginData
	err := c.Bind(&l)
	if err != nil {
		log.Errorf("malformed form: %s", err.Error())
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	err = s.auth.LoginPassword(c, l.Name, l.Password)
	if err != nil {
		log.Errorf("could not authenticate: %s", err.Error())
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	return c.Redirect(http.StatusSeeOther, "/admin")
}

func (s *Server) authRoutes() {
	e := s.echo

	e.GET("/auth/:provider/callback", func(c echo.Context) error {
		user, err := gothic.CompleteUserAuth(
			c.Response(),
			gothic.GetContextWithProvider(c.Request(), c.Param("provider")),
		)
		if err != nil {
			return err
		}

		err = s.saveUser(c, user)
		if err != nil {
			return err
		}

		return c.Redirect(http.StatusSeeOther, "/")
	})

	e.GET("/logout/:provider", func(c echo.Context) error {
		err := gothic.Logout(
			c.Response(),
			gothic.GetContextWithProvider(c.Request(), c.Param("provider")),
		)
		if err != nil {
			return err
		}

		// TODO: move cookie deletion to another place
		cook, err := c.Cookie(accessTokenCookieName)
		if err != nil {
			c.Logger().Debugf("cannot find cookie")
		} else {
			c.Logger().Debugf("deleting cookie")
			cook.Expires = time.Now().Add(-100 * time.Hour)
			cook.MaxAge = -1
			c.SetCookie(cook)
		}

		sess, err := s.sessionStore.Get(c.Request(), sessionName)
		if err == nil {
			sess.Options.MaxAge = -1
			err = sess.Save(c.Request(), c.Response())
			if err != nil {
				e.Logger.Errorf("could not delete session: %s", err.Error())
			}
		}

		return c.Redirect(http.StatusSeeOther, "/")
	})

	e.GET("/auth/:provider", func(c echo.Context) error {
		request := gothic.GetContextWithProvider(c.Request(), c.Param("provider"))

		if user, err := gothic.CompleteUserAuth(c.Response(), request); err == nil {
			err := s.saveUser(c, user)
			if err != nil {
				return err
			}

			return c.Redirect(http.StatusSeeOther, "/")
		}

		gothic.BeginAuthHandler(c.Response(), request)
		return nil
	})
}

func (s *Server) saveUser(c echo.Context, user goth.User) error {
	err := s.auth.LoginGoth(c, user)
	if err != nil {
		return err
	}

	sess, err := s.sessionStore.New(c.Request(), sessionName)
	if err != nil {
		return err
	}

	sess.Values["user"] = user
	sess.Values["provider"] = c.Param("provider")
	sess.Values["id"] = user.UserID

	err = sess.Save(c.Request(), c.Response())
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) callbackURL(provider string) string {
	scheme := "http"
	if s.tlsAddr != "" {
		scheme = "https"
	}

	path := fmt.Sprintf("/auth/%s/callback", provider)

	u := url.URL{
		Scheme: scheme,
		Host:   s.callbackHost,
		Path:   path,
	}

	return u.String()
}

func thumbPath(dataPath string, id int) string {
	return filepath.Join(dataPath, pathThumbs, fmt.Sprintf("%d.png", id))
}

func saveImage(path string, data []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("cannot create thumbnail: %w", err)
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		return fmt.Errorf("cannot write thumbnail: %w", err)
	}

	return nil
}

func idVersion(param string) (int, int, error) {
	var idString, versionString string
	parts := strings.Split(strings.TrimPrefix(param, "#"), ".")
	switch len(parts) {
	case 1:
		idString = parts[0]
	case 2:
		idString = parts[0]
		versionString = parts[1]
	default:
		return 0, 0, ErrInvalidData
	}

	id, err := strconv.Atoi(idString)
	if err != nil {
		return 0, 0, ErrInvalidData
	}

	version, err := strconv.Atoi(versionString)
	if err != nil {
		return 0, 0, ErrInvalidData
	}

	return id, version, nil
}
