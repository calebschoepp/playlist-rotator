package main

import (
	"fmt"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/zmb3/spotify"
)

// Home should be removed
type Home struct {
	Link string
}

func backupMain() {
	auth.SetAuthInfo("1d95de217eb149c9a15bde7ecd9ed724", "b27aedfb22d542119490515a7dfdf2da")
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/callback", redirectHandler)
	fmt.Println("Serving on localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

var redirectURL = "http://localhost:8080/callback"
var stateCookieName = "LoginStateCookie"
var auth = spotify.NewAuthenticator(redirectURL, spotify.ScopeUserReadPrivate, spotify.ScopePlaylistModifyPrivate, spotify.ScopeUserLibraryRead)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	// the redirect URL must be an exact match of a URL you've registered for your application
	// scopes determine which permissions the user is prompted to authorize

	// get the user to this URL - how you do that is up to you
	// you should specify a unique state string to identify the session
	state := "shouldBeRandomString"
	url := auth.AuthURL(state)
	addCookie(w, stateCookieName, state, 30*time.Minute)
	t, _ := template.ParseFiles("./web/home.html")
	t.Execute(w, Home{Link: url})
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	stateCookie, _ := r.Cookie(stateCookieName)
	// use the same state string here that you used to generate the URL
	token, err := auth.Token(stateCookie.Value, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusNotFound)
		return
	}
	// create a client using the specified token
	client := auth.NewClient(token)

	limit := 50
	likedSongs, err := client.CurrentUsersTracksOpt(&spotify.Options{Limit: &limit})
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Couldn't get liked songs", http.StatusInternalServerError)
		return
	}
	tracks := likedSongs.Tracks
	fmt.Println(tracks[0].FullTrack.SimpleTrack.Name)
	t, _ := template.ParseFiles("./web/callback.html")
	t.Execute(w, tracks)
}

func addCookie(w http.ResponseWriter, name, value string, ttl time.Duration) {
	expire := time.Now().Add(ttl)
	cookie := http.Cookie{
		Name:    name,
		Value:   value,
		Expires: expire,
	}
	http.SetCookie(w, &cookie)
}
