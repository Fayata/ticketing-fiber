// handlers/auth.go
package handlers

import (
	"log"
	"time"

	"ticketing-fiber/config"
	"ticketing-fiber/models"
	"ticketing-fiber/utils"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	cfg *config.Config
}

func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{cfg: cfg}
}

// ShowLogin menampilkan halaman login
func (h *AuthHandler) ShowLogin(c *fiber.Ctx) error {
	return c.Render("tickets/login", fiber.Map{
		"title": "Login - Portal Ticketing",
	})
}

// Login proses login user
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	username := c.FormValue("username")
	password := c.FormValue("password")
	rememberMe := c.FormValue("remember_me")

	log.Printf("Login attempt for user: %s", username)

	// Cari user berdasarkan username atau email
	var user models.User
	if err := config.DB.Preload("Groups").
		Where("username = ? OR email = ?", username, username).
		First(&user).Error; err != nil {
		log.Printf("User not found: %s", username)
		return c.Render("tickets/login", fiber.Map{
			"error": "Username atau password salah. Silakan coba lagi.",
			"title": "Login - Portal Ticketing",
		})
	}

	// Cek password
	if !utils.CheckPasswordHash(password, user.Password) {
		log.Printf("Invalid password for user: %s", username)
		return c.Render("tickets/login", fiber.Map{
			"error": "Username atau password salah. Silakan coba lagi.",
			"title": "Login - Portal Ticketing",
		})
	}

	// Cek akses portal
	if !user.HasPortalAccess() {
		log.Printf("User %s doesn't have portal access", username)
		return c.Render("tickets/login", fiber.Map{
			"error": "Akun ini tidak memiliki akses ke dashboard pengguna.",
			"title": "Login - Portal Ticketing",
		})
	}

	// Update last login
	now := time.Now()
	user.LastLogin = &now
	config.DB.Save(&user)

	// Create session
	sess, err := config.Store.Get(c)
	if err != nil {
		log.Printf("Session error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Session error")
	}

	sess.Set("user_id", user.ID)
	sess.Set("username", user.Username)

	// Set session expiry
	if rememberMe == "" {
		sess.SetExpiry(0) // Session expires when browser closes
	} else {
		sess.SetExpiry(14 * 24 * time.Hour) // 2 weeks
	}

	if err := sess.Save(); err != nil {
		log.Printf("Failed to save session: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to save session")
	}

	log.Printf("Login successful for user: %s", username)

	// Check for next parameter
	next := c.FormValue("next")
	if next == "" {
		next = c.Query("next")
	}

	if next != "" {
		return c.Redirect(next)
	}

	return c.Redirect("/dashboard")
}

// ShowRegister menampilkan halaman registrasi
func (h *AuthHandler) ShowRegister(c *fiber.Ctx) error {
	return c.Render("tickets/register", fiber.Map{
		"title": "Registrasi - Portal Ticketing",
	})
}

// Register proses registrasi user baru
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	username := c.FormValue("username")
	email := c.FormValue("email")
	password1 := c.FormValue("password1")
	password2 := c.FormValue("password2")

	// Validasi
	errors := make(map[string]string)

	if username == "" {
		errors["username"] = "Username wajib diisi"
	}
	if email == "" {
		errors["email"] = "Email wajib diisi"
	}
	if password1 == "" {
		errors["password1"] = "Password wajib diisi"
	}
	if password1 != password2 {
		errors["password2"] = "Password tidak cocok"
	}

	// Cek username exists
	var existingUser models.User
	if err := config.DB.Where("username = ?", username).First(&existingUser).Error; err == nil {
		errors["username"] = "Username sudah digunakan"
	}

	// Cek email exists
	if err := config.DB.Where("email = ?", email).First(&existingUser).Error; err == nil {
		errors["email"] = "Email sudah terdaftar"
	}

	if len(errors) > 0 {
		return c.Render("tickets/register", fiber.Map{
			"errors":   errors,
			"username": username,
			"email":    email,
			"title":    "Registrasi - Portal Ticketing",
		})
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(password1)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to hash password")
	}

	// Create user
	user := models.User{
		Username: username,
		Email:    email,
		Password: hashedPassword,
		IsActive: true,
	}

	if err := config.DB.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to create user")
	}

	// Add to Portal Users group
	var portalGroup models.Group
	config.DB.FirstOrCreate(&portalGroup, models.Group{Name: "Portal Users"})
	config.DB.Model(&user).Association("Groups").Append(&portalGroup)

	log.Printf("New user registered: %s", username)

	// Flash message (simplified, Anda bisa gunakan session untuk flash messages)
	return c.Redirect("/login?registered=true")
}

// Logout proses logout user
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	sess, err := config.Store.Get(c)
	if err != nil {
		return c.Redirect("/login")
	}

	// Destroy session
	if err := sess.Destroy(); err != nil {
		log.Printf("Failed to destroy session: %v", err)
	}

	return c.Redirect("/login")
}
