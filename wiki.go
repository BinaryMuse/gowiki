package main

import(
  "net/http"
  "io/ioutil"
  "text/template"
  "regexp"
  "errors"
)

const lenPath = len("/view/")
var templates = template.Must(template.ParseFiles("tmpl/edit.html", "tmpl/view.html"))
var titleChars = "a-zA-Z0-9"
var titleValidator= regexp.MustCompile("^[" + titleChars + "]+$")
var linkRegex = regexp.MustCompile("\\[([" + titleChars + "]+)\\]")
var fileHandler = http.FileServer(http.Dir("public"))

type Page struct {
  Title string
  Body []byte
}

func fileForTitle(title string) string {
  return "data/" + title + ".txt"
}

func replaceWikiLinks(src []byte) []byte {
  stringSrc := string(src)
  stringSrc = stringSrc[1:len(stringSrc)-1]
  return []byte("<a href='/view/" + stringSrc + "'>" + stringSrc + "</a>")
}

func (p *Page) save() error {
  filename := fileForTitle(p.Title)
  return ioutil.WriteFile(filename, p.Body, 0600)
}

func (p *Page) ParseWiki() []byte {
  return linkRegex.ReplaceAllFunc(p.Body, replaceWikiLinks)
}

func loadPage(title string) (*Page, error) {
  filename := fileForTitle(title)
  body, err := ioutil.ReadFile(filename)
  if err != nil {
    return nil, err
  }
  return &Page{Title: title, Body: body}, nil
}

func getTitle(w http.ResponseWriter, r *http.Request) (title string, err error) {
  title = r.URL.Path[lenPath:]
  if !titleValidator.MatchString(title) {
    http.NotFound(w, r)
    err = errors.New("Invalid Page Title")
  }
  return
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
  err := templates.ExecuteTemplate(w, tmpl + ".html", p)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
  p, err := loadPage(title)
  if err != nil {
    http.Redirect(w, r, "/edit/" + title, http.StatusFound)
    return
  }
  renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
  p, err := loadPage(title)
  if err != nil {
    p = &Page{Title: title}
  }
  renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
  body := r.FormValue("body")
  p := &Page{Title: title, Body: []byte(body)}
  err := p.save()
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  http.Redirect(w, r, "/view/" + title, http.StatusFound)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
  if r.URL.Path == "/" {
    http.Redirect(w, r, "/view/Home", http.StatusFound)
  } else {
    fileHandler.ServeHTTP(w, r)
  }
}

func makeHandler(fn func (http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
  return func(w http.ResponseWriter, r *http.Request) {
    title := r.URL.Path[lenPath:]
    if !titleValidator.MatchString(title) {
      http.NotFound(w, r)
      return
    }
    fn(w, r, title)
  }
}

func main() {
  http.HandleFunc("/view/", makeHandler(viewHandler))
  http.HandleFunc("/edit/", makeHandler(editHandler))
  http.HandleFunc("/save/", makeHandler(saveHandler))
  http.HandleFunc("/", rootHandler)
  http.ListenAndServe(":8080", nil)
}
