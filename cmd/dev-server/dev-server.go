package main

import (
	"flag"
	"fmt"
	"github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

const (
	templateFolder  = "./web/templates"
	resourceFolder  = "./web/static"
	localPort       = ":8080"
	refreshInterval = 1 * time.Second
	openOn          = "index"
)

func main() {
	log.Println("dev-server started on port ", localPort)
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
				if info.ModTime().After(fileWatch[file.Name()]) {
					fileWatch[file.Name()] = info.ModTime()
					needsRefresh = true
				}
			}
		}
	}()

	router := mux.NewRouter().StrictSlash(true)
	staticRouter := router.PathPrefix("/static/")
	staticRouter.Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(resourceFolder))))

	fs := flag.NewFlagSet("dev-server", flag.ExitOnError)
	prometheusConfig := prometheus.Flags(fs, "prometheus")
	prometheusApp := prometheus.New(prometheusConfig)

	router.Handle("/metrics", prometheusApp.Handler()).Methods(http.MethodGet)

	router.HandleFunc("/reload", func(wr http.ResponseWriter, req *http.Request) {
		if needsRefresh {
			log.Println("Forcing reload")
			wr.WriteHeader(http.StatusUpgradeRequired)
			return
		}
		// Weird quirk but with an empty response and status code for no content, Chrome still views it
		// as a failed load
		fmt.Fprint(wr, "{}")
	}).Methods(http.MethodGet)
	router.HandleFunc("/reload.js", func(wr http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(wr, fmt.Sprintf(`
		setInterval(function() {
			fetch("/reload")
			.then(function (res) {
				if (res.status == 426) {
					window.location.reload(true);
				}
			})
		}, %d);
		`, refreshInterval.Milliseconds()))
		needsRefresh = false
	}).Methods(http.MethodGet)

	router.HandleFunc("/{template_name}", func(wr http.ResponseWriter, req *http.Request) {

		templateName := mux.Vars(req)["template_name"]
		templatePath := fmt.Sprintf("%s/%s.tpl.html", templateFolder, templateName)
		templateFileContents, err := ioutil.ReadFile(templatePath)
		if err != nil {
			fmt.Fprintf(wr, "Can't read file '%s' - %s", templatePath, err)
			return
		}
		// Load and inject JS
		tmpl, err := template.New(templateName).Parse(
			strings.Replace(string(templateFileContents), "</head>", `<script type="text/javascript" src="/reload.js"></script></head>`, 1))
		if err != nil || tmpl == nil {
			fmt.Fprintf(wr, "Can't find template '%s' - %s", templateName, err)
			return
		}
		_, err = tmpl.ParseGlob(fmt.Sprintf("%s/*", templateFolder))
		if err != nil {
			fmt.Fprintf(wr, "Can't load more template '%s' - %s", templateName, err)
			return
		}
		err = tmpl.ExecuteTemplate(wr, templateName, nil)
		if err != nil {
			fmt.Fprintf(wr, "Can't exec templatePath '%s' - %s", templateName, err)
			return
		}
	}).Methods(http.MethodGet)

	// Disable Caching of results
	router.Use(
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(wr http.ResponseWriter, req *http.Request) {
				wr.Header().Set("Cache-Control", "max-age=0, must-revalidate")
				next.ServeHTTP(wr, req)
			})
		},
	)

	devSrv := http.Server{
		Addr:           localPort,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   120 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Open localhost and start server
	go open(fmt.Sprintf("http://localhost%s/%s", localPort, openOn))
	devSrv.ListenAndServe()
}

// open opens the specified URL in the default browser of the user.
func open(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}
