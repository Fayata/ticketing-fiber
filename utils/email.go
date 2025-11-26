package utils

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"time"

	"ticketing-fiber/config"
)

type EmailService struct {
	cfg *config.Config
}

func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{cfg: cfg}
}

// SendMail mengirim email menggunakan net/smtp standar (Lebih stabil untuk STARTTLS)
func (e *EmailService) SendMail(to []string, subject, body string) error {
	// 1. Setup Alamat Server
	addr := fmt.Sprintf("%s:%d", e.cfg.EmailHost, e.cfg.EmailPort)

	// 2. Setup Header Email (MIME)
	// Penting agar body email terbaca rapi
	headers := make(map[string]string)
	headers["From"] = e.cfg.EmailFrom
	headers["To"] = strings.Join(to, ",")
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/plain; charset=\"utf-8\""

	// Gabungkan header dan body
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	// 3. Mekanisme Retry (Coba Ulang)
	maxRetries := 3
	retryDelay := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		log.Printf("[Email] Percobaan kirim %d/%d ke %v...", i+1, maxRetries, to)

		err := e.sendSMTPSecure(addr, e.cfg.EmailHost, e.cfg.EmailUsername, e.cfg.EmailPassword, to, []byte(message))

		if err == nil {
			log.Printf("✅ Email BERHASIL dikirim ke: %v", to)
			return nil
		}

		log.Printf("⚠️ Gagal kirim (Coba %d): %v", i+1, err)

		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}

	return fmt.Errorf("gagal mengirim email setelah %d percobaan", maxRetries)
}

// sendSMTPSecure menangani koneksi SMTP dengan STARTTLS secara manual
func (e *EmailService) sendSMTPSecure(addr, host, user, password string, to []string, msg []byte) error {
	// A. Connect ke Server (Koneksi Awal Polos)
	// Timeout dialer agar tidak hanging selamanya jika server down
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()

	// B. Lakukan STARTTLS (Upgrade ke Enkripsi)
	// Ini langkah yang sebelumnya gagal karena library lama mencoba SSL duluan
	if ok, _ := c.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true, // Abaikan validasi sertifikat (Solusi error wsarecv/cert)
			ServerName:         host,
		}
		if err = c.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("gagal starttls: %v", err)
		}
	}

	// C. Login / Autentikasi
	// Dilakukan SETELAH koneksi aman (TLS aktif)
	if user != "" && password != "" {
		auth := smtp.PlainAuth("", user, password, host)
		if err = c.Auth(auth); err != nil {
			return fmt.Errorf("gagal auth: %v", err)
		}
	}

	// D. Kirim Data Email
	if err = c.Mail(user); err != nil {
		return err
	}
	for _, t := range to {
		if err = c.Rcpt(t); err != nil {
			return err
		}
	}

	w, err := c.Data()
	if err != nil {
		return err
	}

	_, err = w.Write(msg)
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	return c.Quit()
}

// --- Helper Methods (Tetap sama untuk menjaga kompatibilitas dengan Handler) ---

func (e *EmailService) SendTicketConfirmation(to, username, title string, ticketID uint, department, priority, status, description string) error {
	subject := fmt.Sprintf("[Ticket ID: %d] %s", ticketID, title)

	body := fmt.Sprintf(`Halo %s,

Terima kasih telah menghubungi kami. Tiket Anda telah berhasil dibuat dengan rincian berikut:

ID Tiket  : %d
Judul     : %s
Departemen: %s
Prioritas : %s
Status    : %s

Deskripsi:
%s

---
Tim support kami akan segera meninjau tiket Anda.
Mohon menunggu balasan dari tim support melalui email ini.

Salam,
Tim Support`, username, ticketID, title, department, priority, status, description)

	return e.SendMail([]string{to}, subject, body)
}

func (e *EmailService) SendTicketReply(to, username, title string, ticketID uint, status, replyMessage, replierName string) error {
	subject := fmt.Sprintf("RE: [Ticket ID: %d] %s", ticketID, title)

	body := fmt.Sprintf(`Halo %s,

Tim support kami (%s) telah membalas tiket Anda:

---
%s
---

Detail Tiket:

ID Tiket    : %d
Judul       : %s
Status      : %s

Silakan balas email ini jika ada pertanyaan tambahan.

Salam,
%s
Tim Support`, username, replierName, replyMessage, ticketID, title, status, replierName)

	return e.SendMail([]string{to}, subject, body)
}
