package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/go-martini/martini"
	"github.com/joho/godotenv"
	"github.com/martini-contrib/cors"
	"github.com/martini-contrib/encoder"
)

type Response struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

func main() {
	m := martini.Classic()

	var envs map[string]string

	envs, err := godotenv.Read(".env")

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	m.Use(func(c martini.Context, w http.ResponseWriter, r *http.Request) {
		// Use indentations. &pretty=1
		pretty, _ := strconv.ParseBool(r.FormValue("pretty"))
		// Use null instead of empty object for json &null=1
		null, _ := strconv.ParseBool(r.FormValue("null"))
		// Some content negotiation
		switch r.Header.Get("Content-Type") {
		case "application/xml":
			c.MapTo(encoder.XmlEncoder{PrettyPrint: pretty}, (*encoder.Encoder)(nil))
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		default:
			c.MapTo(encoder.JsonEncoder{PrettyPrint: pretty, PrintNull: null}, (*encoder.Encoder)(nil))
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		}
	})

	m.Use(cors.Allow(&cors.Options{
		AllowOrigins:  []string{"http://localhost*", "http://127.0.0.1*"},
		AllowMethods:  []string{"PUT", "PATCH", "POST", "OPTIONS"},
		AllowHeaders:  []string{"Origin", "X-API-KEY", "Content-Type", "Accept", "Referer", "User-Agent"},
		ExposeHeaders: []string{"Content-Length"},
	}))

	m.Use(func(res http.ResponseWriter, req *http.Request) {
		apiKey := envs["API_KEY"]

		if req.Header.Get("X-API-KEY") != apiKey {
			res.WriteHeader(http.StatusUnauthorized)
		}
	})

	m.Get("/", func(enc encoder.Encoder) (int, []byte) {
		result := Response{"Hello World!", "200"}
		return http.StatusOK, encoder.Must(enc.Encode(result))
	})

	m.Post("/upload/:identifier", func(enc encoder.Encoder, params martini.Params, res http.ResponseWriter, req *http.Request) (int, []byte) {
		status := upload(req, params)
		if status != "200" {
			result := Response{"Error happened during save!", status}
			return http.StatusServiceUnavailable, encoder.Must(enc.Encode(result))
		}

		result := Response{"Saved!", status}
		return http.StatusOK, encoder.Must(enc.Encode(result))
	})

	port := "3000"

	if envs["PORT"] != "" {
		port = envs["PORT"]
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		panic(err)
	}

	go http.Serve(listener, m)
	log.Println("Listening on 0.0.0.0:" + port)

	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGTERM)
	<-sigs
	fmt.Println("SIGTERM, time to shutdown")
	listener.Close()
}

func upload(r *http.Request, params martini.Params) string {
	r.ParseMultipartForm(10 << 20)
	file, handler, err := r.FormFile("file")
	if err != nil {
		fmt.Println("Error Retrieving the File")
		fmt.Println(err)
		return "500"
	}
	defer file.Close()

	fmt.Printf("Uploaded File: %+v\n", handler.Filename)
	fmt.Printf("File Size: %+v\n", handler.Size)
	fmt.Printf("MIME Header: %+v\n", handler.Header)

	// a particular naming pattern
	filename := fmt.Sprintf("upload-%s-*.wav", params["identifier"])
	tempFile, err := ioutil.TempFile("temp-videos", filename)
	if err != nil {
		fmt.Println(err)
		return "500"
	}
	defer tempFile.Close()

	// read all of the contents of our uploaded file into a
	// byte array
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
		return "500"
	}
	// write this byte array to our temporary file
	tempFile.Write(fileBytes)

	return "200"
}
