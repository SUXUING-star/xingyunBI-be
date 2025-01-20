// utils/email.go
package utils

import (
	"bi-backend/config"

	"gopkg.in/gomail.v2"
)

func SendEmail(to, subject, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", config.GlobalConfig.Email.From)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(
		config.GlobalConfig.Email.Host,
		config.GlobalConfig.Email.Port,
		config.GlobalConfig.Email.Username,
		config.GlobalConfig.Email.Password,
	)

	return d.DialAndSend(m)
}
