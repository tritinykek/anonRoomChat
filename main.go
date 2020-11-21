package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"html/template"
	"log"
	"net"
	"net/http"
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

var DBLOGIN = "dbuser"
var DBPASSWORD = "dbpassword"
var DBNAME = "test"

type message struct {
	Id      int
	Message string
}

func createAndOpen(name string) *sql.DB {

	db, err := sql.Open("mysql", DBLOGIN+":"+DBPASSWORD+"@tcp(127.0.0.1:3306)/"+DBNAME)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS " + name + " (`id` integer AUTO_INCREMENT NOT NULL PRIMARY KEY, `message` varchar(1024) NOT NULL)")
	if err != nil {
		panic(err)
	}
	db.Close()

	db, err = sql.Open("mysql", DBLOGIN+":"+DBPASSWORD+"@tcp(127.0.0.1:3306)/"+DBNAME)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	return db
}

func (p *Page) save(w http.ResponseWriter, r *http.Request) error {
	db, err := sql.Open("mysql", DBLOGIN+":"+DBPASSWORD+"@tcp(127.0.0.1:3306)/"+DBNAME)
	defer db.Close()
	if err != nil {
		panic(err)
	}
	createAndOpen(p.Title)
	if err != nil {
		panic(err)
	}
	key := "1234"
	str := getIP(w, r)
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(str))
	str = hex.EncodeToString(h.Sum(nil))[0:6]
	message := string(p.Body)
	fmt.Println("Message: " + strings.Replace(message, "\n", "", -1))
	currentTime := time.Now()

	_, err = db.Exec("INSERT INTO " + p.Title + "(message) VALUES (" + "'" + str + ": " + strings.Replace(message, "\n", "", -1) + " (" + currentTime.Format("2006-01-02 15:04:05 Monday"+")"+"'") + ")")
	return nil
}

func loadPage(title string, w http.ResponseWriter, r *http.Request) (*Page, error) {
	//filename := title + ".txt"
	//b, err := ioutil.ReadFile(filename)
	/*if err != nil {
		return nil, err
	}*/
	//body := ""

	var messages []string
	createAndOpen(title)
	db, err := sql.Open("mysql", DBLOGIN+":"+DBPASSWORD+"@tcp(127.0.0.1:3306)/"+DBNAME)
	defer db.Close()
	if err != nil {
		panic(err)
	}
	res, err := db.Query("SELECT * FROM " + title)

	if err != nil {
		panic(err)
	}

	for res.Next() {
		var msg message
		err := res.Scan(&msg.Id, &msg.Message)
		if err != nil {
			panic(err)
		}
		messages = append(messages, msg.Message)
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
	err := p.save(w, r)

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
	db, err := sql.Open("mysql", DBLOGIN+":"+DBPASSWORD+"@tcp(127.0.0.1:3306)/"+DBNAME)
	defer db.Close()
	if err != nil {
		panic(err)
	}
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))

	log.Fatal(http.ListenAndServe(":8080", nil))

}
