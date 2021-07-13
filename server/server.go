package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/mrdoob/glsl-sandbox/server/store"
)

const (
	pathGallery = "./server/assets/gallery.html"
	perPage     = 50
)

type Template struct {
	templates *template.Template
}

func (t *Template) Render(
	w io.Writer, name string, data interface{}, c echo.Context,
) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

type galleryEffect struct {
	ID      int
	Version int
	Image   string
}

type galleryData struct {
	Effects      []galleryEffect
	IsPrevious   bool
	PreviousPage int
	IsNext       bool
	NextPage     int
}

type Server struct {
	echo     *echo.Echo
	template *Template
	store    store.Store
}

func New(s store.Store) *Server {
	return &Server{
		echo: echo.New(),
		template: &Template{
			templates: template.Must(template.ParseFiles(pathGallery)),
		},
		store: s,
	}
}

func (s *Server) Start() error {
	s.setup()
	return s.echo.Start("localhost:8888")
}

func (s *Server) setup() {
	s.echo.Renderer = s.template
	s.echo.Logger.SetLevel(log.DEBUG)
	s.echo.Use(middleware.Logger())
	s.routes()
}

func (s *Server) routes() {
	s.echo.GET("/", s.indexHandler)
	s.echo.GET("/e", s.effectHandler)
	s.echo.GET("/item/:id", s.itemHandler)

	s.echo.Static("/thumbs", "./data/thumbs")
	s.echo.Static("/css", "./server/assets/css")
	s.echo.Static("/js", "./server/assets/js")
}

func (s *Server) indexHandler(c echo.Context) error {
	pString := c.QueryParam("page")
	if pString == "" {
		pString = "0"
	}
	page, err := strconv.Atoi(pString)
	if err != nil {
		page = 0
	}

	p, err := s.store.Page(page, perPage, false)
	if err != nil {
		return c.String(http.StatusInternalServerError, "error")
	}

	effects := make([]galleryEffect, len(p))
	for i, e := range p {
		effects[i] = galleryEffect{
			ID:      e.ID,
			Version: len(e.Versions) - 1,
			Image:   path.Join("/thumbs", e.ImageName()),
		}
	}

	println(page, len(effects), perPage)
	d := galleryData{
		Effects:      effects,
		IsNext:       len(effects) == perPage,
		NextPage:     page + 1,
		IsPrevious:   page > 0,
		PreviousPage: page - 1,
	}

	err = c.Render(http.StatusOK, "gallery", d)
	if err != nil {
		panic(err)
	}
	return err
	// return c.File("./server/assets/gallery.html")
	// return c.String(http.StatusOK, "index")
}

func (s *Server) effectHandler(c echo.Context) error {
	return c.File("./static/index.html")
}

type itemResponse struct {
	Code   string `json:"code"`
	User   string `json:"user"`
	Parent string `json:"parent,omitempty"`
}

func (s *Server) itemHandler(c echo.Context) error {
	param := c.Param("id")
	var idString, versionString string

	parts := strings.Split(param, ".")
	switch len(parts) {
	case 1:
		idString = parts[0]
	case 2:
		idString = parts[0]
		versionString = parts[1]
	default:
		return c.String(http.StatusBadRequest, "{}")
	}

	id, err := strconv.Atoi(idString)
	if err != nil {
		return c.String(http.StatusBadRequest, "{}")
	}

	version, err := strconv.Atoi(versionString)
	if err != nil {
		return c.String(http.StatusBadRequest, "{}")
	}

	effect, err := s.store.Effect(id)
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
