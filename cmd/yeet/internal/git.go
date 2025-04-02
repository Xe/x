package internal

import (
	"context"
	"flag"
	"os"
	"path/filepath"

	"github.com/Songmu/gitconfig"
	"within.website/x/internal/yeet"
)

var (
	GPGKeyFile     = flag.String("gpg-key-file", gpgKeyFileLocation(), "GPG key file to sign the package")
	GPGKeyID       = flag.String("gpg-key-id", "", "GPG key ID to sign the package")
	GPGKeyPassword = flag.String("gpg-key-password", "", "GPG key password to sign the package")
	UserName       = flag.String("git-user-name", GitUserName(), "user name in Git")
	UserEmail      = flag.String("git-user-email", GitUserEmail(), "user email in Git")
)

const (
	fallbackName  = "Mimi Yasomi"
	fallbackEmail = "mimi@techaro.lol"
)

func gpgKeyFileLocation() string {
	folder, err := os.UserConfigDir()
	if err != nil {
		return ""
	}

	return filepath.Join(folder, "within.website", "x", "yeet", "key.asc")
}

func GitUserName() string {
	name, err := gitconfig.User()
	if err != nil {
		return fallbackName
	}

	return name
}

func GitUserEmail() string {
	email, err := gitconfig.Email()
	if err != nil {
		return fallbackEmail
	}

	return email
}

func GitVersion() string {
	vers, err := yeet.GitTag(context.Background())
	if err != nil {
		panic(err)
	}
	return vers[1:]
}
