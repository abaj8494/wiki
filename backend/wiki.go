package main

import (
	//"fmt"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DATA STRUCTURES
type Page struct {
  Title string
  Body []byte // byte slice. what is expected by the io lib
  Files []string // Array of file names associated with this page
}

// For the index page to display all available pages
type IndexPage struct {
  Pages []string // List of page titles
}

// GLOBAL VARIABLES
var templates = template.Must(template.ParseFiles("edit.html", "view.html", "index.html"))
var validPath = regexp.MustCompile("^/(edit|save|view|upload|delete|delete-file)/([a-zA-Z0-9]+)$")
var filesDir = "./files" // Directory to store uploaded files
var persistentDir = "/app/persistence" // Directory to store persistent storage

// enableCORS adds CORS headers to allow requests from the frontend
func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "https://abaj.ai")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// corsMiddleware wraps handlers with CORS support
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

/*
func handler(w http.ResponseWriter, r *http.Request) {
  fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:]) //slicing drops the leading /
}
*/

func getTitle(w http.ResponseWriter, r *http.Request) (string, error) {
  m := validPath.FindStringSubmatch(r.URL.Path)
  if m == nil {
    http.NotFound(w, r)
    return "", errors.New("invalid Page Title")
  }
  return m[2], nil // the title is the second subexpression.
}


func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
  err := templates.ExecuteTemplate(w, tmpl+".html",p)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }
}

// FUNCTION LITERALS and CLOSURES
/*
The closure returned by makeHandler is a function that takes an http.ResponseWriter and http.Request (in other words, an http.HandlerFunc). The closure extracts the title from the request path, and validates it with the validPath regexp. If the title is invalid, an error will be written to the ResponseWriter using the http.NotFound function. If the title is valid, the enclosed handler function fn will be called with the ResponseWriter, Request, and title as arguments.
*/
func makeHandler(fn func (http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
  // called a closure. fn is one of the xxxxHandlers
  return func(w http.ResponseWriter, r *http.Request) {
    enableCORS(w)
    m := validPath.FindStringSubmatch(r.URL.Path)
    if m == nil {
      http.NotFound(w, r)
      return
    }
    fn(w, r, m[2])
  }
}


func viewHandler(w http.ResponseWriter,r *http.Request, title string) {
  /* transcended with the closures
  title, err := getTitle(w, r)
  if err != nil {
    return
  }*/
  p, err := loadPage(title)
  if err != nil {
    http.Redirect(w, r, "/edit/"+title, http.StatusFound)
    return
  }
  /* new and improved version above of the below: error handling!
  title := r.URL.Path[len("/view/"):]
  p, _ := loadPage(title)
  */
  renderTemplate(w, "view", p)
  //fmt.Fprintf(w, "<h1>%s</h1><div>%s</div>", p.Title, p.Body)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
  /* same regex error handling as in other handlers.
  title := r.URL.Path[len("/edit/"):]
  p, err := loadPage(title)
  */
  /* closures. preventing code repetition.
  title, err := getTitle(w, r)
  if err != nil {
    return
  }*/
  p, err := loadPage(title)
  if err != nil {
    p = &Page{Title: title}
  }
  renderTemplate(w, "edit", p)
  /* Hardcoded html:
  fmt.Fprintf(w, "<h1>Editing %s</h1>"+
    "<form action=\"/save/%s\" method=\"POST\">"+
    "<textarea name=\"body\">%s</textarea><br>"+
    "<input type=\"submit\" value=\"Save\">"+
    "</form>",
    p.Title, p.Title, p.Body)
  */
  /* Code repetition:
  t, _ := template.ParseFiles("edit.html")
  t.Execute(w,p)
  */
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
  /* closured;
  title, err := getTitle(w, r)
  if err != nil {
    return
  }*/
  //title := r.URL.Path[len("/save/"):]
  body := r.FormValue("body")
  p, err := loadPage(title)
  if err != nil {
    p = &Page{Title: title, Body: []byte(body)}
  } else {
    p.Body = []byte(body)
  }
  err = p.save()
  if err != nil {
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
  }
  
  // Immediately back up the file after saving
  go BackupWikiFiles()
  
  http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

// uploadHandler handles file uploads for a specific page
func uploadHandler(w http.ResponseWriter, r *http.Request, title string) {
  if r.Method != "POST" {
    http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    return
  }

  // Ensure files directory exists
  pageDirPath := filepath.Join(filesDir, title)
  if err := os.MkdirAll(pageDirPath, 0755); err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }

  // Parse multipart form, 10 << 20 specifies maximum upload of 10 MB files
  r.ParseMultipartForm(10 << 20)
  
  // Get file from form
  file, handler, err := r.FormFile("file")
  if err != nil {
    http.Error(w, "Error retrieving file: "+err.Error(), http.StatusBadRequest)
    return
  }
  defer file.Close()

  // Create file in the server
  filePath := filepath.Join(pageDirPath, handler.Filename)
  dst, err := os.Create(filePath)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  defer dst.Close()

  // Copy file contents
  if _, err := io.Copy(dst, file); err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }

  // Update page to include the file
  p, err := loadPage(title)
  if err != nil {
    p = &Page{Title: title, Body: []byte{}, Files: []string{handler.Filename}}
  } else {
    // Check if file is already in the list
    found := false
    for _, f := range p.Files {
      if f == handler.Filename {
        found = true
        break
      }
    }
    if !found {
      p.Files = append(p.Files, handler.Filename)
    }
  }
  err = p.save()
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  
  // Immediately back up the files after uploading
  go BackupWikiFiles()

  w.WriteHeader(http.StatusOK)
  w.Write([]byte("File uploaded successfully"))
}

// apiGetPageHandler returns page content as JSON
func apiGetPageHandler(w http.ResponseWriter, r *http.Request) {
  enableCORS(w)
  title := r.URL.Query().Get("title")
  if title == "" {
    http.Error(w, "Missing title parameter", http.StatusBadRequest)
    return
  }

  p, err := loadPage(title)
  if err != nil {
    p = &Page{Title: title}
  }

  w.Header().Set("Content-Type", "application/json")
  w.Write([]byte(`{"title":"` + p.Title + `","body":"` + string(p.Body) + `","files":["` + 
    join(p.Files, `","`) + `"]}`))
}

// Helper function to join strings with a separator
func join(s []string, sep string) string {
  if len(s) == 0 {
    return ""
  }
  result := s[0]
  for _, v := range s[1:] {
    result += sep + v
  }
  return result
}

func (p *Page) save() error {
  filename := p.Title + ".txt"
  
  // Write page data
  err := os.WriteFile(filename, p.Body, 0600)
  if err != nil {
    return err
  }
  
  // Write files list if there are any
  if len(p.Files) > 0 {
    filesListFilename := p.Title + ".files.txt"
    filesContent := join(p.Files, "\n")
    err = os.WriteFile(filesListFilename, []byte(filesContent), 0600)
    if err != nil {
      return err
    }
  }
  
  return nil
}

func loadPage(title string) (*Page, error) {
  filename := title + ".txt"
  body, err := os.ReadFile(filename)
  if err != nil {
    // Try to restore from persistent storage if file not found
    restoreErr := RestoreWikiFile(title)
    if restoreErr == nil {
      // Successfully restored, try reading again
      body, err = os.ReadFile(filename)
      if err != nil {
        return nil, err
      }
    } else {
      return nil, err
    }
  }
  
  // Load files list if it exists
  filesListFilename := title + ".files.txt"
  var files []string
  filesContent, err := os.ReadFile(filesListFilename)
  if err == nil && len(filesContent) > 0 {
    files = regexp.MustCompile(`\r?\n`).Split(string(filesContent), -1)
  }
  
  return &Page{Title: title, Body: body, Files: files}, nil
}

func getAllPages() []string {
  // Get all .txt files (wiki pages)
  files, err := filepath.Glob("*.txt")
  if err != nil {
    return []string{}
  }
  
  // Extract titles (remove .txt extension)
  titles := make([]string, 0, len(files))
  for _, file := range files {
    // Skip .files.txt metafiles
    if !strings.HasSuffix(file, ".files.txt") {
      title := strings.TrimSuffix(file, ".txt")
      titles = append(titles, title)
    }
  }
  
  return titles
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
  if r.URL.Path != "/" {
    http.NotFound(w, r)
    return
  }
  
  // Get all available pages
  pages := getAllPages()
  indexPage := &IndexPage{Pages: pages}
  
  err := templates.ExecuteTemplate(w, "index.html", indexPage)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }
}

// deleteHandler handles the deletion of a wiki page
func deleteHandler(w http.ResponseWriter, r *http.Request, title string) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Delete the main text file
	filename := title + ".txt"
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		http.Error(w, fmt.Sprintf("Error deleting page: %v", err), http.StatusInternalServerError)
		return
	}

	// Delete files list if it exists
	filesListFilename := title + ".files.txt"
	os.Remove(filesListFilename) // Ignore errors as the file might not exist

	// Delete the files directory if it exists
	pageDirPath := filepath.Join(filesDir, title)
	if _, err := os.Stat(pageDirPath); err == nil {
		if err := os.RemoveAll(pageDirPath); err != nil {
			log.Printf("Error removing files directory for %s: %v", title, err)
		}
	}

	// Also remove from persistence if possible
	persistentPath := filepath.Join(persistentDir, filename)
	os.Remove(persistentPath) // Ignore errors
	
	persistentFilesList := filepath.Join(persistentDir, filesListFilename)
	os.Remove(persistentFilesList) // Ignore errors
	
	persistentFilesDir := filepath.Join(persistentDir, "files", title)
	os.RemoveAll(persistentFilesDir) // Ignore errors

	http.Redirect(w, r, "/", http.StatusFound)
}

// deleteFileHandler handles the deletion of a specific file attachment
func deleteFileHandler(w http.ResponseWriter, r *http.Request, title string) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fileName := r.FormValue("filename")
	if fileName == "" {
		http.Error(w, "Missing filename parameter", http.StatusBadRequest)
		return
	}

	// First, remove the file from the filesystem
	filePath := filepath.Join(filesDir, title, fileName)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		http.Error(w, fmt.Sprintf("Error deleting file: %v", err), http.StatusInternalServerError)
		return
	}

	// Then, update the page's files list
	p, err := loadPage(title)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Remove the file from the Files slice
	var updatedFiles []string
	for _, f := range p.Files {
		if f != fileName {
			updatedFiles = append(updatedFiles, f)
		}
	}
	p.Files = updatedFiles

	// Save the updated page
	if err := p.save(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Immediately back up the files after deletion
	go BackupWikiFiles()

	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func main() {
  // Create files directory if it doesn't exist
  if err := os.MkdirAll(filesDir, 0755); err != nil {
    log.Fatal(err)
  }

  // Set up file watcher to periodically backup wiki files
  SetupFileWatcher()

  // Set up static file server for uploaded files
  fileServer := http.FileServer(http.Dir(filesDir))
  http.Handle("/files/", http.StripPrefix("/files/", corsMiddleware(fileServer)))

  // Set up static file server for icon files
  iconServer := http.FileServer(http.Dir("./icon"))
  http.Handle("/icon/", http.StripPrefix("/icon/", corsMiddleware(iconServer)))
  
  // Serve favicon.ico directly from the icon directory
  http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "./icon/favicon.ico")
  })

  // Root handler
  http.HandleFunc("/", rootHandler)

  // API endpoints
  http.HandleFunc("/api/page", apiGetPageHandler)

  // Traditional wiki endpoints
  http.HandleFunc("/view/", makeHandler(viewHandler))
  http.HandleFunc("/edit/", makeHandler(editHandler))
  http.HandleFunc("/save/", makeHandler(saveHandler))
  http.HandleFunc("/upload/", makeHandler(uploadHandler))
  http.HandleFunc("/delete/", makeHandler(deleteHandler))
  http.HandleFunc("/delete-file/", makeHandler(deleteFileHandler))
  
  log.Println("Starting server on http://localhost:21313")
  log.Fatal(http.ListenAndServe(":21313", nil))
}
