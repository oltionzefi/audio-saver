package main

import (
	"fmt"
	"io"
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
	Message string `json:"login"`
	Code    string `json:"url"`
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

	m.Post("/save/:file", func(enc encoder.Encoder, params martini.Params, res http.ResponseWriter, req *http.Request) (int, []byte) {
		code := UploadSound(req, params)
		if code != "200" {
			result := Response{"Error happened during save!", code}
			return http.StatusServiceUnavailable, encoder.Must(enc.Encode(result))
		}

		result := Response{"Saved!", code}
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

func UploadSound(r *http.Request, params martini.Params) string {
	file, _, err := r.FormFile("audio")

	if err != nil {
		return "500"
	}

	defer file.Close()

	filename := fmt.Sprintf("upload/%s.wav", params["file"])
	out, err := os.Create(filename)

	if err != nil {
		return "500"
	}

	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		return "500"
	}

	return "200"
}
