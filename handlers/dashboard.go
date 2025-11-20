// handlers/dashboard.go
package handlers

import (
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

	// Menggunakan konstanta status dari models/ticket.go
	// Pastikan string di database SAMA PERSIS dengan yang di query ini
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

	return c.Render("tickets/dashboard", fiber.Map{
		"title":               "Dashboard - Portal Ticketing",
		"user":                user,
		"waiting_tickets":     waitingCount,
		"in_progress_tickets": inProgressCount,
		"closed_tickets":      closedCount,
		"total_tickets":       totalCount,
		"recent_tickets":      recentTickets,
		// Placeholder empty slices untuk menyamai view Django yang mengirim list kosong
		"announcements":    []interface{}{},
		"popular_articles": []interface{}{},
		"unread_count":     0,
	})
}
