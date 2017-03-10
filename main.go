package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	mgo "gopkg.in/mgo.v2"

	"github.com/dkundathagard/chat/trace"
	"github.com/gorilla/mux"
	"github.com/stretchr/gomniauth"
	"github.com/stretchr/gomniauth/providers/google"
	"github.com/stretchr/signature"
)

// set the active Avatar implementation
var avatars Avatar = TryAvatars{
	UseFileSystemAvatar,
	UseAuthAvatar,
	UseGravatarAvatar,
}

func main() {
	session, err := mgo.Dial("localhost")
	if err != nil {
		log.Fatalln("Could not connect to local MongoDB database.")
	}
	defer session.Close()
	db := session.DB("chaton")
	addr := flag.String("addr", ":8080", "The addr of the application.")
	flag.Parse()
	gomniauth.SetSecurityKey(signature.RandomKey(64))
	gomniauth.WithProviders(
		google.New(
			os.Getenv("GOOGLE_ID"),
			os.Getenv("GOOGLE_SECRET"),
			"http://localhost:8080/auth/callback/google",
		),
	)
	r := newRoom("RoomA", db.C("RoomA"))
	r.tracer = trace.New(os.Stdout)
	router := mux.NewRouter()
	router.HandleFunc("/", handleIndex)
	router.Handle("/chat", MustAuth(&templateHandler{filename: "chat.html"}))
	router.Handle("/login", &templateHandler{filename: "login.html"})
	router.HandleFunc("/auth/{action}/{provider}", loginHandler)
	router.Handle("/room", r)
	router.Handle("/upload", MustAuth(&templateHandler{filename: "upload.html"}))
	router.HandleFunc("/uploader", uploaderHandler)
	router.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:   "auth",
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})
		w.Header().Set("Location", "/chat")
		w.WriteHeader(http.StatusTemporaryRedirect)
	})
	http.Handle("/avatars/",
		http.StripPrefix("/avatars/",
			http.FileServer(http.Dir("./avatars"))))
	// get the room going
	go r.run()
	// start web server
	log.Println("Starting web server on", *addr)
	if err := http.ListenAndServe(*addr, router); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
