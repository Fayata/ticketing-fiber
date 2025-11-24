package handlers

import (
	"log"
	"strings"

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

	// Get flash messages from query params (simplified)
	successMsg := c.Query("success")
	errorMsg := c.Query("error")

	data := fiber.Map{
		"title": "Settings - Portal Ticketing",
		"user":  user,
	}

	if successMsg != "" {
		data["success"] = successMsg
	}

	if errorMsg != "" {
		data["error"] = errorMsg
	}

	return h.renderSettingsPage(c, data)
}

// UpdateProfile update profil user
func (h *SettingsHandler) UpdateProfile(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	username := strings.TrimSpace(c.FormValue("username"))
	email := strings.TrimSpace(c.FormValue("email"))
	firstName := strings.TrimSpace(c.FormValue("first_name"))
	lastName := strings.TrimSpace(c.FormValue("last_name"))

	// Validasi
	errors := make(map[string]string)

	if username == "" {
		errors["username"] = "Username wajib diisi"
	}
	if email == "" {
		errors["email"] = "Email wajib diisi"
	}

	// Check username exists (exclude current user)
	var existingUser models.User
	if username != "" {
		if err := config.DB.Where("username = ? AND id != ?", username, user.ID).First(&existingUser).Error; err == nil {
			errors["username"] = "Username sudah digunakan"
		}
	}

	// Check email exists (exclude current user)
	if email != "" {
		if err := config.DB.Where("email = ? AND id != ?", email, user.ID).First(&existingUser).Error; err == nil {
			errors["email"] = "Email sudah terdaftar"
		}
	}

	if len(errors) > 0 {
		return h.renderSettingsPage(c, fiber.Map{
			"errors":        errors,
			"form_username": username,
			"form_email":    email,
			"form_first":    firstName,
			"form_last":     lastName,
		})
	}

	// Update user
	user.Username = username
	user.Email = email
	user.FirstName = firstName
	user.LastName = lastName

	if err := config.DB.Save(user).Error; err != nil {
		log.Printf("Failed to update user: %v", err)
		return h.renderSettingsPage(c, fiber.Map{
			"errors": map[string]string{
				"__all__": "Gagal memperbarui profil. Silakan coba lagi.",
			},
			"form_username": username,
			"form_email":    email,
			"form_first":    firstName,
			"form_last":     lastName,
		})
	}

	// Update session username
	sess, _ := config.Store.Get(c)
	sess.Set("username", username)
	sess.Save()

	log.Printf("Profile updated for user: %s", username)

	return c.Redirect("/settings?success=Profil berhasil diperbarui")
}

// ChangePassword mengubah password user
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
		errors["new_password2"] = "Konfirmasi password baru wajib diisi"
	}

	if len(errors) > 0 {
		return h.renderSettingsPage(c, fiber.Map{
			"errors": errors,
		})
	}

	// Check old password
	if !utils.CheckPasswordHash(oldPassword, user.Password) {
		return h.renderSettingsPage(c, fiber.Map{
			"errors": map[string]string{
				"old_password": "Password lama tidak sesuai",
			},
		})
	}

	// Check new passwords match
	if newPassword1 != newPassword2 {
		return h.renderSettingsPage(c, fiber.Map{
			"errors": map[string]string{
				"new_password2": "Password baru tidak cocok",
			},
		})
	}

	// Validate new password length
	if len(newPassword1) < 8 {
		return h.renderSettingsPage(c, fiber.Map{
			"errors": map[string]string{
				"new_password1": "Password minimal 8 karakter",
			},
		})
	}

	// Hash new password
	hashedPassword, err := utils.HashPassword(newPassword1)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		return c.Redirect("/settings?error=Gagal mengubah password")
	}

	// Update password
	user.Password = hashedPassword
	if err := config.DB.Save(user).Error; err != nil {
		log.Printf("Failed to update password: %v", err)
		return h.renderSettingsPage(c, fiber.Map{
			"errors": map[string]string{
				"__all__": "Gagal mengubah password. Silakan coba lagi.",
			},
		})
	}

	log.Printf("Password changed for user: %s", user.Username)

	return c.Redirect("/settings?success=Password berhasil diubah")
}

func (h *SettingsHandler) renderSettingsPage(c *fiber.Ctx, data fiber.Map) error {
	if data == nil {
		data = fiber.Map{}
	}
	if data["title"] == nil {
		data["title"] = "Settings - Portal Ticketing"
	}
	if data["page_title"] == nil {
		data["page_title"] = "Pengaturan Akun"
	}
	if data["page_subtitle"] == nil {
		data["page_subtitle"] = "Kelola informasi profil dan keamanan akun Anda"
	}
	if data["template_name"] == nil {
		data["template_name"] = "tickets/settings"
	}
	return c.Render("tickets/settings", addBaseData(c, data))
}
