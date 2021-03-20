package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-martini/martini"
	"github.com/joho/godotenv"
)

func main() {
	m := martini.Classic()

	var envs map[string]string

	envs, err := godotenv.Read(".env")

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	m.Use(func(res http.ResponseWriter, req *http.Request) {
		apiKey := envs["API_KEY"]

		if req.Header.Get("X-API-KEY") != apiKey {
			res.WriteHeader(http.StatusUnauthorized)
		}
	})

	m.Get("/", func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
		res.Write([]byte("Hello World!"))
	})

	m.Get("/save/:file", func(params martini.Params, res http.ResponseWriter, req *http.Request) {
		code := UploadSound(req, params)
		if code != "200" {
			res.WriteHeader(http.StatusServiceUnavailable)
			res.Write([]byte("Error happened during save!"))
		}

		res.WriteHeader(http.StatusOK)
		res.Write([]byte("Saved!"))
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

	defer file.Close()

	if err != nil {
		return "500"
	}

	filename := fmt.Sprintf("upload/%s.wav", params["file"])
	out, err := os.Create(filename)

	defer out.Close()

	if err != nil {
		return "500"
	}

	_, err = io.Copy(out, file)
	if err != nil {
		return "500"
	}

	return "200"
}
