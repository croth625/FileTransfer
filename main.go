package main

import (
	"fmt"
	"net/http"
	"os"
	"io"
	"strings"
	"errors"
	"encoding/json"
)

type config struct{
	UserPath string `json:"userPath"`
	Subfolder string `json:"subfolder"`
	Port int32 `json:"port"`
	MaxUpload int64 `json:"maxUpload"`
	UploadUnit string `json:"uploadUnits"`
}

var Config config

func main(){
	//Create Config
	Config = config{UserPath:"C:\\Users\\", 
	Subfolder:"\\", 
	Port:80, 
	MaxUpload:1, UploadUnit:"GB"}

	if _, err := os.Stat("config.json"); errors.Is(err, os.ErrNotExist) {
		//file does not exists, create new file from default
		data, err := json.Marshal(Config)
		if err != nil{
			fmt.Printf("%v", err)
		}
		err = os.WriteFile("config.json", data, 0666)
		if err != nil{
			fmt.Printf("%v", err)
		}
	}else{
		//file exists, read config from file
		data, err := os.ReadFile("config.json")
		if err != nil{
			fmt.Printf("%v", err)
		}
		err = json.Unmarshal(data, &Config)
		if err != nil{
			fmt.Printf("%v", err)
		}
	}
	//Convert MaxUpload to bytes
	units := []string{"tb", "gb", "mb", "kb", ""}
	unitsFull := []string{"terrabyte", "gigabyte", "megabyte", "kilobyte", ""}
	for i:=0; i<len(units)-1; i++{
		if strings.ToLower(Config.UploadUnit) == units[i] || strings.ToLower(Config.UploadUnit) == unitsFull[i]{
			Config.MaxUpload *= 1000
			Config.UploadUnit = units[i+1]
		}
	}
	fmt.Printf("Max Upload: %v\n",Config.MaxUpload)
	//start Server
	router := http.NewServeMux()
	server := &http.Server{
		Handler: router,
		Addr:    fmt.Sprintf(":%v", Config.Port),
	}
	addRoutes(router, "")

	err := server.ListenAndServe()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
}

func addRoutes(router *http.ServeMux, folder string) {
	router.Handle("POST "+folder+"/", http.HandlerFunc(RoutePost))
	router.Handle("POST "+folder+"/upload", http.HandlerFunc(RouteUpload))
	router.Handle("GET "+folder+"/", http.HandlerFunc(RouteGet))
	router.Handle("GET "+folder+"/logout", http.HandlerFunc(RouteLogout))
}

func httpRespondWithText(w http.ResponseWriter, code int, payload string, header string) {
	if header != "" {
		w.Header().Add("Content-Type", header)
	}
	w.WriteHeader(code)
	w.Write([]byte(payload))
}

func sendCookie(w http.ResponseWriter, name string, value string, age int) {
	cookie := http.Cookie{
		Name:   name,
		Value:  value,
		Path:   "/",
		MaxAge: age, //seconds
	}
	http.SetCookie(w, &cookie)
}

func readCookie(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func RouteGet(w http.ResponseWriter, r *http.Request){//"/"
	user := readCookie(r, "user")
	if user == ""{
		serveLogin(w, "Login Form")
	}else{
		serveUploadForm(w, user, "")
	}
}

func serveUploadForm(w http.ResponseWriter, user, logText string){
	sendCookie(w, "user", user, 3600*24*30)//refreshes cookie to 30 days
	text:= "<h1>Hello "+ user + `</h1><button onclick="location.href='/logout'">Logout</button>` +
		`<form action="/upload" method="post" enctype="multipart/form-data">
		<input type="file" id="myFile" name="filename" multiple>
		<input type="submit" value="Upload">
		</form><p>`+ logText +"</p>"

	httpRespondWithText(w, 200, text, "text/html")
}

func serveLogin(w http.ResponseWriter, topMessage string){
	text := `<h1>`+ topMessage +`</h1></br>
			<form action="/" method="post">
				<label>Username:</label><input id="user" name="user">
				<button type="submit">Login</button>
			</form>`
	httpRespondWithText(w, 200, text, "text/html")
}

func RoutePost(w http.ResponseWriter, r *http.Request){//"/"
	user := ""
	r.ParseMultipartForm(1000)
	for k, v := range r.Form {
		if k =="user"{user = v[0]}
	}
	if user == ""{
		serveLogin(w, "Login Unsuccessful")
	}else{
		if _, err := os.Stat(Config.UserPath + user); err != nil {
			if os.IsNotExist(err){
				serveLogin(w, "User not found")
				return
			}
		}
		serveUploadForm(w, user, "")
	}
}

func RouteLogout(w http.ResponseWriter, r *http.Request){//"/logout"
	sendCookie(w, "user", "", 1)
	http.Redirect(w, r, "/", 302)
}

func RouteUpload(w http.ResponseWriter, r *http.Request){
	user := readCookie(r, "user")
	fileSaved := 0
	text := ""
	r.ParseMultipartForm(Config.MaxUpload)
	
	if user == ""{
		serveLogin(w, "Login Form")
	}else{
		form := r.MultipartForm
		if form == nil || form.File == nil {
			return
		}

		for _, files := range form.File {
			for _, fileHeader := range files {
				// Open uploaded file
				src, err := fileHeader.Open()
				if err != nil {
					text += "Failed to open uploaded file: " + err.Error() +"</br>"
				}
				defer src.Close()

				// Create a local file to save uploaded data
				fileName := Config.UserPath + user + Config.Subfolder + fileHeader.Filename
				if _, err := os.Stat(fileName); !errors.Is(err, os.ErrNotExist) {// check that file does not exist
					temp := strings.Split(fileName,".")
					for i:=1;;i++{
						tempFileName := strings.Join(temp[0:len(temp)-1],".") + fmt.Sprintf("(%d).", i) + temp[len(temp)-1] //Adds (i) before last .*
						if _, err := os.Stat(tempFileName); errors.Is(err, os.ErrNotExist){//check that file does not exist
							fileName = tempFileName
						}
					}
				}
				dst, err := os.Create(fileName)
				if err != nil {
					text += "Failed to create file: " + err.Error() + "</br>"
				}
				defer dst.Close()

				// Copy file contents
				if _, err = io.Copy(dst, src); err != nil {
					text += "Failed to save file: " + err.Error() + "</br>"
					return
				}

				text += fmt.Sprintf("Saved file %s.</br>", fileName)
				fileSaved++
			}
		}
		serveUploadForm(w, user, fmt.Sprintf("%d", fileSaved) + " Files Saved.</br>" + text)
	}
}
