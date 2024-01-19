package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"time"
)

func GetCC(UID int) (email string, err error) {
	for _, user := range Config.Users {
		if user.UID == UID {
			return user.Email, nil
		}
	}
	return "", errors.New("user not found")
}

func SendSMTP(UID int, MID int, ts time.Time, Content string) (err error) {
	to_address, err := GetCC(UID)
	if err != nil {
		slog.Error("Failed to get email address", "error", err)
		return err
	}

	var conn net.Conn
	address := net.JoinHostPort(Config.Smtp.Host, Config.Smtp.Port)
	if Config.Smtp.TLS {
		conn, err = tls.Dial("tcp", address, &tls.Config{ServerName: Config.Smtp.Host, InsecureSkipVerify: Config.Smtp.Insecure})
	} else {
		conn, err = net.Dial("tcp", address)
	}
	if err != nil {
		slog.Error("Failed to connect to smtp server", "error", err)
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, Config.Smtp.Host)
	if err != nil {
		slog.Error("Failed to create smtp client", "error", err)
		return err
	}
	defer client.Close()

	if Config.Smtp.STARTTLS {
		client.StartTLS(&tls.Config{ServerName: Config.Smtp.Host, InsecureSkipVerify: Config.Smtp.Insecure})
	}

	// Authentication
	auth := smtp.PlainAuth("", Config.Smtp.From, Config.Smtp.Password, Config.Smtp.Host)
	if err = client.Auth(auth); err != nil {
		slog.Error("Failed to auth smtp client", "error", err)
		return err
	}

	// Send to&from
	// To && From
	if err = client.Mail(Config.Smtp.From); err != nil {
		slog.Error("Failed to send FROM", "error", err, "from", Config.Smtp.From)
		return err
	}

	if err = client.Rcpt(to_address); err != nil {
		slog.Error("Failed to send FROM", "error", err, "to", to_address)
		return err
	}

	// Send body
	// Data
	w, err := client.Data()
	if err != nil {
		slog.Error("Failed to send DATA", "error", err)
		return err
	}

	subject := "Memos 定时提醒" + " - " + Content
	body := "Memos 定时提醒" + "\r\n时间：" + ts.Format("2006-01-02 15:04:05") + "\r\n备忘编号：" + strconv.Itoa(MID) + "\r\n内容：" + Content
	from_content := mail.Address{Name: "Memo Reminders", Address: Config.Smtp.From}
	to_content := mail.Address{Name: "", Address: to_address}

	// Setup headers
	headers := make(map[string]string)
	headers["From"] = from_content.String()
	headers["To"] = to_content.String()
	headers["Subject"] = subject
	headers["Content-Type"] = "text/plain; charset=UTF-8"

	// Setup message
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body
	slog.Info("Send email", "from", Config.Smtp.From, "to", to_address, "subject", subject, "server", Config.Smtp.Host, "msg", message)

	_, err = w.Write([]byte(message))
	if err != nil {
		slog.Error("Failed to write DATA", "error", err)
	}
	w.Close()
	return nil
}

func MergeSlice(s1 []string, s2 []string) []string {
	slice := make([]string, len(s1)+len(s2))
	copy(slice, s1)
	copy(slice[len(s1):], s2)
	return slice
}
