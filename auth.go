package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/stretchr/gomniauth"
	authcommon "github.com/stretchr/gomniauth/common"
	"github.com/stretchr/objx"
)

// ChatUser represents a user of the chat room.
type ChatUser interface {
	UniqueID() string
	AvatarURL() string
}

type chatUser struct {
	authcommon.User
	uniqueID string
}

func (u chatUser) UniqueID() string {
	return u.uniqueID
}

// MustAuth wraps an http.Handler, making it require authentication
func MustAuth(handler http.Handler) http.Handler {
	return &authHandler{next: handler}
}

type authHandler struct {
	next http.Handler
}

func (h *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("auth"); err == http.ErrNoCookie || cookie.Value == "" {
		// not authenticated
		w.Header().Set("Location", "/login")
		w.WriteHeader(http.StatusTemporaryRedirect)
		return
	} else if err != nil {
		// some other error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// success - call next handler
	h.next.ServeHTTP(w, r)
}

// loginHandler handles the third-party login process.
// format: /auth/{action}/{provider}
func loginHandler(w http.ResponseWriter, r *http.Request) {
	seqs := strings.Split(r.URL.Path, "/")
	if len(seqs) < 4 {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "URL %s not supported: too few arguments to auth", r.URL.Path)
		return
	}
	action := seqs[2]
	provider := seqs[3]
	switch action {
	case "login":
		provider, err := gomniauth.Provider(provider)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error when trying to get provider %s: %s",
				provider, err), http.StatusBadRequest)
			return
		}
		loginUrl, err := provider.GetBeginAuthURL(nil, nil)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error when trying to GetBeginAuthURL for %s :%s",
				provider, err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Location", loginUrl)
		w.WriteHeader(http.StatusTemporaryRedirect)
	case "callback":
		provider, err := gomniauth.Provider(provider)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error when trying to get provider %s: %s",
				provider, err), http.StatusBadRequest)
			return
		}
		creds, err := provider.CompleteAuth(objx.MustFromURLQuery(r.URL.RawQuery))
		if err != nil {
			http.Error(w, fmt.Sprintf("Error when trying to complete auth for %s, %s",
				provider, err), http.StatusInternalServerError)
			return
		}
		user, err := provider.GetUser(creds)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error when trying to get user from %s, %s",
				provider, err), http.StatusInternalServerError)
			return
		}
		chatUser := &chatUser{User: user}
		m := md5.New()
		io.WriteString(m, strings.ToLower(user.Email()))
		userId := fmt.Sprintf("%x", m.Sum(nil))
		avatarURL, err := avatars.GetAvatarURL(chatUser)
		if err != nil {
			log.Fatalln("Error when trying to GetAvatarURL", "-", err)
		}
		authCookieValue := objx.New(map[string]interface{}{
			"userid":     userId,
			"name":       user.Name(),
			"avatar_url": avatarURL,
		}).MustBase64()
		http.SetCookie(w, &http.Cookie{
			Name:  "auth",
			Value: authCookieValue,
			Path:  "/",
		})
		w.Header().Set("Location", "/chat")
		w.WriteHeader(http.StatusTemporaryRedirect)
	default:
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Auth action %s not supported", action)
	}
}
