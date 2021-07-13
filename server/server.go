package server

import (
	"html/template"
	"io"
	"net/http"
	"path"
	"strconv"

	"github.com/labstack/echo/v4"
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
	s.routes()
}

func (s *Server) routes() {
	s.echo.GET("/", s.indexHandler)
	s.echo.Static("/thumbs", "./data/thumbs")
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
