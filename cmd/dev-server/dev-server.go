package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

const (
	templateFolder = "./web/templates"
)

func main() {
	needsRefresh := false

	// Folder watcher
	go func() {
		fileWatch := map[string]time.Time{}
		for {
			time.Sleep(3 * time.Second)
			f, err := os.Open(templateFolder)
			if err != nil {
				log.Fatal("Couldnt open templateFolder", err)
			}
			files, _ := f.Readdir(-1)
			f.Close()

			for _, file := range files {
				info, _ := os.Stat(f.Name() + "/" + file.Name())
				if err != nil {
					// TODO: handle errors (e.g. file not found)
					log.Fatal("Couldnt stat file", err)
				}
				if fileWatch[file.Name()] != info.ModTime() {
					fileWatch[file.Name()] = info.ModTime()
					needsRefresh = true
				}
			}
		}
	}()

	router := mux.NewRouter().StrictSlash(true)
	staticRouter := router.PathPrefix("/static/")
	staticRouter.Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static"))))
	router.HandleFunc("/reload", func(wr http.ResponseWriter, req *http.Request) {
		if needsRefresh {
			log.Println("Forcing reload")
			wr.WriteHeader(http.StatusUpgradeRequired)
		}
	}).Methods(http.MethodGet)
	router.HandleFunc("/reload.js", func(wr http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(wr, `
		setInterval(function() {
			fetch("/reload")
			.then(function (response) {
				return response;
			})
			.then(function (res) {
				if (res.status == 426) {
					window.location.reload(true);
				}
			})
		}, 1000);
		`)
		needsRefresh = false
	}).Methods(http.MethodGet)

	router.HandleFunc("/{template_name}", func(wr http.ResponseWriter, req *http.Request) {

		templateName := mux.Vars(req)["template_name"]
		templatePath := fmt.Sprintf("./web/templates/%s.tpl.html", templateName)
		templateFileContents, err := ioutil.ReadFile(templatePath)
		if err != nil {
			fmt.Fprintf(wr, "Can't read file '%s' - %s", templatePath, err)
			return
		}
		// Load and inject JS
		tmpl, err := template.New(templateName).Parse(strings.Replace(string(templateFileContents), "</head>", `<script type="text/javascript" src="/reload.js"></script>
		</head>`, 1))
		if err != nil || tmpl == nil {
			fmt.Fprintf(wr, "Can't find template '%s' - %s", templateName, err)
			return
		}
		err = tmpl.ExecuteTemplate(wr, templateName, nil)
		if err != nil {
			fmt.Fprintf(wr, "Can't exec templatePath '%s' - %s", templateName, err)
			return
		}
	}).Methods(http.MethodGet)

	devSrv := http.Server{
		Addr:           ":8080",
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   120 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	devSrv.ListenAndServe()
}
