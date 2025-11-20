package middleware

import (
	"ticketing-fiber/config"
	"ticketing-fiber/models"

	"github.com/gofiber/fiber/v2"
)

// PortalUserRequired middleware untuk memastikan user memiliki akses portal
func PortalUserRequired(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	if !user.HasPortalAccess() {
		return c.Status(fiber.StatusForbidden).Render("tickets/error", fiber.Map{
			"error": "Akses ini khusus untuk akun pengguna portal.",
		})
	}

	return c.Next()
}

// GuestOnly middleware untuk halaman yang hanya boleh diakses user yang belum login
func GuestOnly(c *fiber.Ctx) error {
	sess, err := config.Store.Get(c)
	if err == nil {
		userID := sess.Get("user_id")
		if userID != nil {
			return c.Redirect("/dashboard")
		}
	}

	return c.Next()
}

// SetUserLocals middleware untuk set user info ke semua template
func SetUserLocals(c *fiber.Ctx) error {
	sess, err := config.Store.Get(c)
	if err == nil {
		userID := sess.Get("user_id")
		if userID != nil {
			var user models.User
			if err := config.DB.Preload("Groups").First(&user, userID).Error; err == nil {
				c.Locals("user", &user)
				c.Locals("authenticated", true)

				// Count active tickets
				var activeCount int64
				config.DB.Model(&models.Ticket{}).
					Where("created_by_id = ? AND status != ?", user.ID, models.StatusClosed).
					Count(&activeCount)
				c.Locals("active_tickets_count", activeCount)

				return c.Next()
			}
		}
	}

	c.Locals("authenticated", false)
	c.Locals("active_tickets_count", 0)
	return c.Next()
}
