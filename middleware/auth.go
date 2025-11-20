package middleware

import (
	"ticketing-fiber/config"
	"ticketing-fiber/models"

	"github.com/gofiber/fiber/v2"
)

// AuthRequired middleware untuk memastikan user sudah login
func AuthRequired(c *fiber.Ctx) error {
	sess, err := config.Store.Get(c)
	if err != nil {
		return c.Redirect("/login")
	}

	userID := sess.Get("user_id")
	if userID == nil {
		return c.Redirect("/login")
	}

	// Load user dari database
	var user models.User
	if err := config.DB.Preload("Groups").First(&user, userID).Error; err != nil {
		sess.Destroy()
		return c.Redirect("/login")
	}

	// Set user ke locals untuk diakses di handler
	c.Locals("user", &user)
	c.Locals("authenticated", true)

	return c.Next()
}
