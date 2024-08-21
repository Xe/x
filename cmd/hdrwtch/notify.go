package main

import (
	"strings"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

func (s *Server) messageUser(user *TelegramUser, message string) error {
	var sb strings.Builder

	sb.WriteString(message)

	msg := tu.Message(tu.ID(user.ID), sb.String())
	msg.ParseMode = telego.ModeMarkdown

	if _, err := s.tg.SendMessage(msg); err != nil {
		return err
	}

	return nil
}
