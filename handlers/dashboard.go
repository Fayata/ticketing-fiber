// handlers/dashboard.go
package handlers

import (
	"fmt"

	"ticketing-fiber/config"
	"ticketing-fiber/models"

	"github.com/gofiber/fiber/v2"
)

type DashboardHandler struct {
	cfg *config.Config
}

func NewDashboardHandler(cfg *config.Config) *DashboardHandler {
	return &DashboardHandler{cfg: cfg}
}

func (h *DashboardHandler) ShowDashboard(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	var waitingCount, inProgressCount, closedCount, totalCount int64

	// Menggunakan Status dari model (WAITING, IN_PROGRESS, CLOSED)
	config.DB.Model(&models.Ticket{}).
		Where("created_by_id = ? AND status = ?", user.ID, models.StatusWaiting).
		Count(&waitingCount)

	config.DB.Model(&models.Ticket{}).
		Where("created_by_id = ? AND status = ?", user.ID, models.StatusInProgress).
		Count(&inProgressCount)

	config.DB.Model(&models.Ticket{}).
		Where("created_by_id = ? AND status = ?", user.ID, models.StatusClosed).
		Count(&closedCount)

	config.DB.Model(&models.Ticket{}).
		Where("created_by_id = ?", user.ID).
		Count(&totalCount)

	var recentTickets []models.Ticket
	config.DB.Preload("Department").
		Preload("Replies").
		Where("created_by_id = ?", user.ID).
		Order("created_at DESC").
		Limit(5).
		Find(&recentTickets)

	return c.Render("tickets/dashboard", addBaseData(c, fiber.Map{
		"title":               "Dashboard - Portal Ticketing",
		"page_title":          "Dashboard",
		"page_subtitle":       fmt.Sprintf("Selamat datang kembali, %s!", user.GetFullName()),
		"nav_active":          "dashboard",
		"template_name":       "tickets/dashboard",
		"waiting_tickets":     waitingCount,
		"in_progress_tickets": inProgressCount,
		"closed_tickets":      closedCount,
		"total_tickets":       totalCount,
		"recent_tickets":      recentTickets,
		"announcements":       []interface{}{},
		"popular_articles":    []interface{}{},
		"unread_count":        0,
	}))
}
