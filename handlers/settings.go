// handlers/settings.go
package handlers

import (
	"log"

	"ticketing-fiber/config"
	"ticketing-fiber/models"
	"ticketing-fiber/utils"

	"github.com/gofiber/fiber/v2"
)

type SettingsHandler struct {
	cfg *config.Config
}

func NewSettingsHandler(cfg *config.Config) *SettingsHandler {
	return &SettingsHandler{cfg: cfg}
}

// ShowSettings menampilkan halaman settings
func (h *SettingsHandler) ShowSettings(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	return c.Render("tickets/settings", fiber.Map{
		"title": "Settings - Portal Ticketing",
		"user":  user,
	})
}

// UpdateProfile proses update profil user
func (h *SettingsHandler) UpdateProfile(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	username := c.FormValue("username")
	email := c.FormValue("email")
	firstName := c.FormValue("first_name")
	lastName := c.FormValue("last_name")

	// Validasi
	errors := make(map[string]string)

	if username == "" {
		errors["username"] = "Username wajib diisi"
	}
	if email == "" {
		errors["email"] = "Email wajib diisi"
	}

	// Cek username exists (selain user saat ini)
	var existingUser models.User
	if err := config.DB.Where("username = ? AND id != ?", username, user.ID).First(&existingUser).Error; err == nil {
		errors["username"] = "Username sudah digunakan"
	}

	// Cek email exists (selain user saat ini)
	if err := config.DB.Where("email = ? AND id != ?", email, user.ID).First(&existingUser).Error; err == nil {
		errors["email"] = "Email sudah terdaftar"
	}

	if len(errors) > 0 {
		return c.Render("tickets/settings", fiber.Map{
			"title":  "Settings - Portal Ticketing",
			"user":   user,
			"errors": errors,
		})
	}

	// Update user
	user.Username = username
	user.Email = email
	user.FirstName = firstName
	user.LastName = lastName

	if err := config.DB.Save(user).Error; err != nil {
		log.Printf("Failed to update profile: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to update profile")
	}

	log.Printf("Profile updated for user: %s", username)

	return c.Render("tickets/settings", fiber.Map{
		"title":   "Settings - Portal Ticketing",
		"user":    user,
		"success": "Profil berhasil diupdate",
	})
}

// ChangePassword proses ganti password
func (h *SettingsHandler) ChangePassword(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	oldPassword := c.FormValue("old_password")
	newPassword1 := c.FormValue("new_password1")
	newPassword2 := c.FormValue("new_password2")

	// Validasi
	errors := make(map[string]string)

	if oldPassword == "" {
		errors["old_password"] = "Password lama wajib diisi"
	}
	if newPassword1 == "" {
		errors["new_password1"] = "Password baru wajib diisi"
	}
	if newPassword2 == "" {
		errors["new_password2"] = "Konfirmasi password wajib diisi"
	}
	if newPassword1 != newPassword2 {
		errors["new_password2"] = "Password tidak cocok"
	}

	// Cek password lama
	if oldPassword != "" && !utils.CheckPasswordHash(oldPassword, user.Password) {
		errors["old_password"] = "Password lama salah"
	}

	if len(errors) > 0 {
		return c.Render("tickets/settings", fiber.Map{
			"title":  "Settings - Portal Ticketing",
			"user":   user,
			"errors": errors,
		})
	}

	// Hash password baru
	hashedPassword, err := utils.HashPassword(newPassword1)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to hash password")
	}

	// Update password
	user.Password = hashedPassword
	if err := config.DB.Save(user).Error; err != nil {
		log.Printf("Failed to update password: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to update password")
	}

	log.Printf("Password changed for user: %s", user.Username)

	return c.Render("tickets/settings", fiber.Map{
		"title":   "Settings - Portal Ticketing",
		"user":    user,
		"success": "Password berhasil diubah",
	})
}

