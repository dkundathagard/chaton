package main

import (
	"errors"
	"io/ioutil"
	"path"
)

// ErrNoAvatar is the error that is returned when the Avatar instance is unable
// to provide an avatar URL.
var ErrNoAvatarURL = errors.New("chat: Unable to get an avatar url")

// Avatar represents types capable of representing user profile pictures.
type Avatar interface {
	// GetAvatarURL gets the avatar URL for the specified client,
	// or returns an error if something goes wrong.
	// ErrNoAvatarURL is returned if the object is unable to get
	// a URL for the specified client.
	GetAvatarURL(ChatUser) (string, error)
}

// TryAvatars represents the different ways of accessing a user's avatars
type TryAvatars []Avatar

// GetAvatarURL attempts to get the URL for a user's avatar from all of the
// available avatar URL retrieval methods
func (a TryAvatars) GetAvatarURL(u ChatUser) (string, error) {
	for _, avatar := range a {
		if url, err := avatar.GetAvatarURL(u); err == nil {
			return url, nil
		}
	}
	return "", ErrNoAvatarURL
}

// AuthAvatar is represents an avatar provided by the OAuth2 provider
type AuthAvatar struct{}

var UseAuthAvatar AuthAvatar

func (AuthAvatar) GetAvatarURL(u ChatUser) (string, error) {
	url := u.AvatarURL()
	if len(url) == 0 {
		return "", ErrNoAvatarURL
	}
	return url, nil
}

type GravatarAvatar struct{}

var UseGravatarAvatar GravatarAvatar

func (GravatarAvatar) GetAvatarURL(u ChatUser) (string, error) {
	return "https://www.gravatar.com/avatar/" + u.UniqueID(), nil
}

type FileSystemAvatar struct{}

var UseFileSystemAvatar FileSystemAvatar

func (FileSystemAvatar) GetAvatarURL(u ChatUser) (string, error) {
	files, err := ioutil.ReadDir("avatars")
	if err != nil {
		return "", ErrNoAvatarURL
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if match, _ := path.Match(u.UniqueID()+"*", file.Name()); match {
			return "/avatars/" + file.Name(), nil
		}
	}
	return "", ErrNoAvatarURL
}
