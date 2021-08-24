package main

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"os/exec"
)

var tplI = template.Must(template.ParseFiles("index.html"))
var tplR = template.Must(template.ParseFiles("resp.html"))

type InputSource struct {
	Input string
	Resp  string
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tplI.Execute(w, nil)
}

func respHandler(w http.ResponseWriter, r *http.Request) {
	u, err := url.Parse(r.URL.String())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
		return
	}

	params := u.Query()
	input := params.Get("input")
	fmt.Println("Input is: ", input)

	// This is an anti-pattern. Do not use in production.
	cmd := exec.Command("sh", "-c", "ping -c 1 "+input)
	stdout, err := cmd.Output()

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Print(string(stdout))

	resp := InputSource{input, string(stdout)}

	tplR.Execute(w, resp)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("assets"))
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))

	mux.HandleFunc("/resp", respHandler)

	mux.HandleFunc("/", indexHandler)
	http.ListenAndServe(":"+port, mux)
}
