package handlers

import (
	"github.com/gofiber/fiber/v2"
)

func addBaseData(c *fiber.Ctx, data fiber.Map) fiber.Map {
	if data == nil {
		data = fiber.Map{}
	}

	if data["title"] == nil {
		data["title"] = "Portal Ticketing"
	}

	if user := c.Locals("user"); user != nil {
		data["user"] = user
	}

	if count := c.Locals("active_tickets_count"); count != nil {
		data["active_tickets_count"] = count
	} else if _, ok := data["active_tickets_count"]; !ok {
		data["active_tickets_count"] = 0
	}

	return data
}
