package main

import (
	"embed"
	"flag"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
)

var (
	basePath = flag.String("path", "./", "base file path")
	port     = flag.String("port", ":9081", "listen port")
	username = flag.String("user", "", "username")

	exts = []string{".gif", ".png", ".jpg", ".tif", ".tiff", ".zip", ".rar", ".cbz", ".cbr", ".bmp", ".pdf", ".cgt"}
)

func init() {
	log.SetFlags(log.LstdFlags | log.Llongfile | log.LUTC)
	flag.Parse()
}

type PageInfo struct {
	DirEntries  []MyEntry
	CurrentPath string
}

type MyEntry struct {
	DirEntry os.DirEntry
	Path     string
	BookDate int64
}

func handler(resp http.ResponseWriter, req *http.Request) {
	log.Println(req)
	if http.MethodGet != req.Method && http.MethodHead != req.Method {
		log.Panicln("only support get/head method")
	}
	path := req.URL.Path
	if path == "" {
		path = "/"
	} else {
		path = filepath.Clean(path)
	}
	if path == "/favicon.ico" {
		favicon(resp)
		return
	}
	absPath, err := filepath.Abs(filepath.Join(*basePath, path))
	if err != nil {
		log.Panicln(err)
	}
	pathInfo, err := os.Stat(absPath)
	if err != nil {
		log.Panicln(pathInfo)
	}
	pathFile, err := os.Open(absPath)
	defer pathFile.Close()
	if err != nil {
		log.Panicln(err)
	}
	if !pathInfo.IsDir() {
		download(resp, pathFile)
		return
	}
	entries, err := pathFile.ReadDir(-1)
	if err != nil {
		log.Panicln(err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return isDirValue(entries[i].IsDir()) < isDirValue(entries[j].IsDir())
	})
	page := PageInfo{DirEntries: make([]MyEntry, 0, 32), CurrentPath: path}
	for _, e := range entries {
		if e.Name() == "." || e.Name() == ".." {
			continue
		} else {
			if !e.IsDir() {
				ex := filepath.Ext(e.Name())
				found := false
				for _, i := range exts {
					if ex == i {
						found = true
					}
				}
				if !found {
					continue
				}
			}
			inf, err := e.Info()
			if err != nil {
				log.Panicln(err)
			}
			page.DirEntries = append(page.DirEntries, MyEntry{e, filepath.Join(path, e.Name()), inf.ModTime().Unix()})
		}
	}
	render(resp, page)
}

func download(resp http.ResponseWriter, path *os.File) {
	_, err := io.Copy(resp, path)
	if err != nil {
		log.Panicln(err)
	}
}

//go:embed index.gohtml images.png
var embedFS embed.FS

func favicon(resp http.ResponseWriter) {
	favi, err := embedFS.Open("images.png")
	if err != nil {
		log.Panicln(err)
	}
	_, err = io.Copy(resp, favi)
	if err != nil {
		log.Panicln(err)
	}
}

func render(resp http.ResponseWriter, info PageInfo) {
	tpl, err := template.New("index.gohtml").ParseFS(embedFS, "index.gohtml")
	if err != nil {
		log.Panicln(err)
	}
	err = tpl.Execute(resp, info)
	if err != nil {
		log.Panicln(err)
	}
}

func isDirValue(v bool) int {
	if v {
		return 0
	}
	return 1
}

func main() {
	err := http.ListenAndServe("0.0.0.0"+*port, http.HandlerFunc(handler))
	if err != nil {
		log.Panicln(err)
	}
}
