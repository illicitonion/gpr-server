package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
)

var (
	port            = flag.Int("port", 8081, "Port")
	baseNodeModules = flag.String("base_node_modules", "", "Base node_modules dir")
	outputDir       = flag.String("output_dir", "", "Output dir")
)

func handle(w http.ResponseWriter, req *http.Request) {
	reject := func(err error) {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
	}

	if !regexp.MustCompile(`^/\d*$`).MatchString(req.URL.Path) {
		reject(fmt.Errorf("Bad PR"))
		return
	}

	resp, err := http.Get("https://api.github.com/repos/Kegsay/github-pull-review/pulls" + req.URL.Path)
	if err != nil {
		reject(err)
		return
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		reject(err)
		return
	}

	var r pr
	if err := json.Unmarshal(b, &r); err != nil {
		reject(err)
		return
	}

	if !regexp.MustCompile("^[0-9a-f]{40}$").MatchString(r.Head.SHA) {
		reject(fmt.Errorf("Bad SHA: " + r.Head.SHA))
		return
	}

	if r.User.Login != "illicitonion" {
		reject(fmt.Errorf("Wrong user: " + r.User.Login))
		return
	}

	dst := *outputDir + "/" + r.Head.SHA

	if _, err := os.Stat(dst); os.IsNotExist(err) {
		dir, err := ioutil.TempDir("", "repo")
		if err != nil {
			reject(err)
		}
		if err := run(dir, "git", "clone", "https://github.com/illicitonion/github-pull-review.git", "."); err != nil {
			reject(err)
			return
		}
		if err := run(dir, "git", "checkout", r.Head.SHA); err != nil {
			reject(err)
			return
		}
		if *baseNodeModules != "" {
			if err := run(dir, "ln", "-s", *baseNodeModules); err != nil {
				reject(err)
				return
			}
		}
		if err := run(dir, "npm", "install"); err != nil {
			reject(err)
			return
		}
		if err := run(dir, "npm", "run", "build"); err != nil {
			reject(err)
			return
		}
		if err := run(dir, "/bin/bash", "-c", "mkdir -p "+dst+" && mv build/* "+dst+"/"); err != nil {
			reject(err)
			return
		}
	}

	w.Header().Set("Location", "https://review.rocks/shas/"+r.Head.SHA)
	w.WriteHeader(302)
}

func main() {
	flag.Parse()

	if *outputDir == "" {
		log.Fatalf("Must set output_dir")
	}

	http.HandleFunc("/", handle)
	http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", *port), nil)
}

func run(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = dir
	return cmd.Run()
}

type pr struct {
	Head ref
	User user
}

type ref struct {
	SHA string
}

type user struct {
	Login string
}
