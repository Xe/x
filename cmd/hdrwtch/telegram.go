package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"time"
)

type TelegramUser struct {
	ID         string `json:"id" gorm:"primaryKey"`
	FirstName  string `json:"first_name" gorm:"not null"`
	LastName   string `json:"last_name" gorm:"not null"`
	Username   string `json:"username" gorm:"index,not null"`
	PhotoURL   string `json:"photo_url" gorm:"not null"`
	AuthDate   string `json:"auth_date" gorm:"not null"`
	IsAdmin    bool   `json:"is_admin" gorm:"not null"`
	ProbeLimit int    `json:"probe_limit" gorm:"not null"`
}

type ctxKey int

const (
	ctxKeyTelegramUser ctxKey = iota
)

func getTelegramUser(ctx context.Context) (*TelegramUser, bool) {
	val, ok := ctx.Value(ctxKeyTelegramUser).(*TelegramUser)
	return val, ok
}

func (s *Server) loggedIn(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := s.getTelegramUserData(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, ctxKeyTelegramUser, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func checkTelegramAuthorization(botToken string, authData map[string]string) (*TelegramUser, error) {
	checkHash := authData["hash"]
	delete(authData, "hash")

	var dataCheckArr []string
	for key, value := range authData {
		dataCheckArr = append(dataCheckArr, key+"="+value)
	}
	sort.Strings(dataCheckArr)
	dataCheckString := ""
	for _, item := range dataCheckArr {
		dataCheckString += item + "\n"
	}
	dataCheckString = dataCheckString[:len(dataCheckString)-1]

	secretKey := sha256.Sum256([]byte(botToken))
	h := hmac.New(sha256.New, secretKey[:])
	h.Write([]byte(dataCheckString))
	hash := hex.EncodeToString(h.Sum(nil))

	if hash != checkHash {
		return nil, errors.New("data is NOT from Telegram")
	}

	authDate, err := strconv.ParseInt(authData["auth_date"], 10, 64)
	if err != nil {
		return nil, err
	}
	if time.Now().Unix()-authDate > 86400 {
		return nil, errors.New("data is outdated")
	}

	user := &TelegramUser{
		ID:         authData["id"],
		FirstName:  authData["first_name"],
		LastName:   authData["last_name"],
		Username:   authData["username"],
		PhotoURL:   authData["photo_url"],
		AuthDate:   authData["auth_date"],
		IsAdmin:    authData["username"] == "miamorecadenza",
		ProbeLimit: 5,
	}

	return user, nil
}

func (s *Server) saveTelegramUserData(w http.ResponseWriter, r *http.Request, user *TelegramUser) error {
	session, err := s.store.Get(r, "telegram-session")
	if err != nil {
		return err
	}

	session.Values["user_id"] = user.ID
	return session.Save(r, w)
}

func (s *Server) getTelegramUserData(r *http.Request) (*TelegramUser, bool) {
	session, err := s.store.Get(r, "telegram-session")
	if err != nil {
		return nil, false
	}

	userID, ok := session.Values["user_id"].(string)
	if !ok {
		return nil, false
	}

	user := &TelegramUser{ID: userID}
	if err := s.dao.db.First(user).WithContext(r.Context()).Error; err != nil {
		return nil, false
	}

	return user, true
}
