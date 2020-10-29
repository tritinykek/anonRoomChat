package main

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

func getIP(w http.ResponseWriter, req *http.Request) string {
	ip, port, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		//return nil, fmt.Errorf("userip: %q is not IP:port", req.RemoteAddr)

		fmt.Println("userip: %q is not IP:port", req.RemoteAddr)
	}

	// This will only be defined when site is accessed via non-anonymous proxy
	// and takes precedence over RemoteAddr
	// Header.Get is case-insensitive
	forward := req.Header.Get("X-Forwarded-For")

	fmt.Println("IP: %s", ip)
	fmt.Println("Port: %s", port)
	fmt.Println("Forwarded for: %s", forward)
	return ip
}

type Page struct {
	Title    string
	Body     []byte
	Messages []string
}

func (p *Page) save(w http.ResponseWriter, r *http.Request) error {
	filename := p.Title + ".txt"
	filename1, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	key := "1234"
	str := getIP(w, r)
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(str))
	str = hex.EncodeToString(h.Sum(nil))[0:6]
	message := string(p.Body)
	fmt.Println(strings.Replace(message,"\n","",-1))
	currentTime := time.Now()
	filename1.WriteString(str + ": " + strings.Replace(message,"\n","",-1)+" ("+currentTime.Format("2006-01-02 15:04:05 Monday")+")"+"\n")


	return nil
}

func loadPage(title string, w http.ResponseWriter, r *http.Request) (*Page, error) {
	filename := title + ".txt"
	//b, err := ioutil.ReadFile(filename)
	/*if err != nil {
		return nil, err
	}*/
	//body := ""

	var messages []string

	file, err := os.Open(filename)
	if err != nil {
		file, err = os.Create(filename)
		log.Panic(err)

	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

		message := scanner.Text()
		messages = append(messages, message)
	}

	return &Page{Title: title, Messages: messages}, nil
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {

	getIP(w, r)

	p, err := loadPage(title, w, r)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	getIP(w, r)
	p, err := loadPage(title, w, r)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save(w,r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

var templates = template.Must(template.ParseFiles("edit.html", "view.html"))

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

func main() {
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))

	log.Fatal(http.ListenAndServe(":8080", nil))

}