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

	// Get flash messages from query params (simplified)
	successMsg := c.Query("success")
	errorMsg := c.Query("error")

	return c.Render("tickets/settings", fiber.Map{
		"title":       "Settings - Portal Ticketing",
		"user":        user,
		"success_msg": successMsg,
		"error_msg":   errorMsg,
	})
}

// UpdateProfile update profil user
func (h *SettingsHandler) UpdateProfile(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	username := c.FormValue("username")
	email := c.FormValue("email")
	firstName := c.FormValue("first_name")
	lastName := c.FormValue("last_name")

	// Validasi
	if username == "" || email == "" {
		return c.Redirect("/settings?error=Username dan email wajib diisi")
	}

	// Check username exists (exclude current user)
	var existingUser models.User
	if err := config.DB.Where("username = ? AND id != ?", username, user.ID).First(&existingUser).Error; err == nil {
		return c.Redirect("/settings?error=Username sudah digunakan")
	}

	// Check email exists (exclude current user)
	if err := config.DB.Where("email = ? AND id != ?", email, user.ID).First(&existingUser).Error; err == nil {
		return c.Redirect("/settings?error=Email sudah terdaftar")
	}

	// Update user
	user.Username = username
	user.Email = email
	user.FirstName = firstName
	user.LastName = lastName

	if err := config.DB.Save(user).Error; err != nil {
		log.Printf("Failed to update user: %v", err)
		return c.Redirect("/settings?error=Gagal memperbarui profil")
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
	if oldPassword == "" || newPassword1 == "" || newPassword2 == "" {
		return c.Redirect("/settings?error=Semua field password wajib diisi")
	}

	// Check old password
	if !utils.CheckPasswordHash(oldPassword, user.Password) {
		return c.Redirect("/settings?error=Password lama tidak sesuai")
	}

	// Check new passwords match
	if newPassword1 != newPassword2 {
		return c.Redirect("/settings?error=Password baru tidak cocok")
	}

	// Validate new password length
	if len(newPassword1) < 8 {
		return c.Redirect("/settings?error=Password minimal 8 karakter")
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
		return c.Redirect("/settings?error=Gagal mengubah password")
	}

	log.Printf("Password changed for user: %s", user.Username)

	return c.Redirect("/settings?success=Password berhasil diubah")
}
