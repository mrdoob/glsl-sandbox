# glsl-sandbox

## Server development

### Setup

* Fork repository
* Download repository and create new development branch:

```
$ git clone git@github.com:<your user>/glsl-sandbox
$ cd glsl-sandbox
$ git checkout -b <feature name> go
```

* Download and uncompress test data:

```
$ curl -O https://downloads.zooloo.org/glslsandbox-data.tar.gz
$ tar xvf glslsandbox-data.tar.gz
```

* Build server binary needs go compiler:

```
$ go build ./server/cmd/glslsandbox
```

* Alternatively you can download and uncompress the binary in the repository directory from https://github.com/mrdoob/glsl-sandbox/releases/latest

* Run server:

```
$ ./glslsandbox
```

* The first time it starts it creates an admin user and the credentials are printed.

* The server should be accessible on http://localhost:8888

* Admin interface is on http://localhost:8888/admin

### Template and javascript modifications

The server reloads templates and assets on each query. This eases the development as you can modify the files and changes will take effect reloading the page.

There's only one template that is used for both the gallery (index) and admin page. The file is `server/assets/gallery.html` and uses go language templates. You can find more information about its syntax here:

* https://gohugo.io/templates/introduction/
* https://pkg.go.dev/text/template
* https://pkg.go.dev/html/template

Currently the page receives this data:

```go
// galleryEffect has information about each effect displayed in the gallery.
type galleryEffect struct {
	// ID is the effect identifyier.
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
	PreviousPage int
	// IsNext is true if there is a next page.
	IsNext bool
	// NextPage is the next page number.
	NextPage int
	// Admin is true when accessing "/admin" path.
	Admin bool
}
```

This is, `galleryData` for the page and `galleryEffect` for each effect. For example, to print all the effect IDs you can use:

```html
<ul>
{{ range .Effects }}
    <li>{{ .ID }}</li>
{{ end }}
<ul>
```

The following directories are accessible from the server and can be modified if needed:

* `server/assets/css` -> `/css`
* `server/assets/js` -> `/js`

By default the data files are read from `./data`. This path can be hanged with the environment variable `DATA_PATH`. For example:

```
$ DATA_PATH=/my/data/directory ./glslsandbox
```

The data directory contains the sqlite database (`glslsandbox.db`) and the thumbnails (`thumbs` directory).
