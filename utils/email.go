package utils

import (
	"fmt"
	"log"

	"ticketing-fiber/config"

	"gopkg.in/gomail.v2"
)

type EmailService struct {
	cfg *config.Config
}

func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{cfg: cfg}
}

func (e *EmailService) SendMail(to []string, subject, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", e.cfg.EmailFrom)
	m.SetHeader("To", to...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	d := gomail.NewDialer(
		e.cfg.EmailHost,
		e.cfg.EmailPort,
		e.cfg.EmailUsername,
		e.cfg.EmailPassword,
	)

	if err := d.DialAndSend(m); err != nil {
		log.Printf("Failed to send email: %v", err)
		return err
	}

	log.Printf("✅ Email sent successfully to: %v", to)
	return nil
}

func (e *EmailService) SendTicketConfirmation(to, username, title string, ticketID uint, department, priority, status, description string) error {
	subject := fmt.Sprintf("[Ticket ID: %d] %s", ticketID, title)

	body := fmt.Sprintf(`Halo %s,
	Terima kasih telah menghubungi kami. Tiket Anda telah berhasil dibuat dengan rincian berikut:
	ID Tiket: %d
	Judul: %s
	Departemen: %s
	Prioritas: %s
	Status: %s

	Deskripsi:
	%s

	---
	Tim support kami akan segera meninjau tiket Anda dan memberikan balasan.
	Mohon menunggu balasan dari tim support melalui email ini.

	Jika Anda memiliki pertanyaan lebih lanjut, silakan hubungi kami.

	Salam,
	Tim Support`, username, ticketID, title, department, priority, status, description)

	return e.SendMail([]string{to}, subject, body)
}

func (e *EmailService) SendTicketReply(to, username, title string, ticketID uint, status, replyMessage, replierName string) error {
	subject := fmt.Sprintf("RE: [Ticket ID: %d] %s", ticketID, title)

	body := fmt.Sprintf(`Halo %s,Tim support kami (%s) telah membalas tiket Anda:

	---
	%s
	---

	Detail Tiket:
	ID Tiket: %d
	Judul: %s
	Status: %s

	Jika Anda ingin memberikan tanggapan atau informasi tambahan,
	silakan balas email ini atau hubungi tim support kami.

	Salam,
	%s
	Tim Support`, username, replierName, replyMessage, ticketID, title, status, replierName)

	return e.SendMail([]string{to}, subject, body)
}
