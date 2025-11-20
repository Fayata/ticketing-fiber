package handlers

import (
	"fmt"
	"log"
	"strconv"

	"ticketing-fiber/config"
	"ticketing-fiber/models"
	"ticketing-fiber/utils"

	"github.com/gofiber/fiber/v2"
)

type TicketHandler struct {
	cfg          *config.Config
	emailService *utils.EmailService
}

func NewTicketHandler(cfg *config.Config, emailService *utils.EmailService) *TicketHandler {
	return &TicketHandler{
		cfg:          cfg,
		emailService: emailService,
	}
}

// ShowCreateTicket menampilkan form create ticket
func (h *TicketHandler) ShowCreateTicket(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	// Check if departments exist
	var departmentCount int64
	config.DB.Model(&models.Department{}).Count(&departmentCount)
	if departmentCount == 0 {
		return c.Render("tickets/setup_error", fiber.Map{
			"title": "Error Konfigurasi",
		})
	}

	// Get all departments
	var departments []models.Department
	config.DB.Find(&departments)

	return c.Render("tickets/create_ticket", fiber.Map{
		"title":       "Kirim Tiket Baru - Portal Ticketing",
		"departments": departments,
		"user":        user,
	})
}

// CreateTicket proses pembuatan ticket baru
func (h *TicketHandler) CreateTicket(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	title := c.FormValue("title")
	description := c.FormValue("description")
	replyToEmail := c.FormValue("reply_to_email")
	priority := c.FormValue("priority")
	departmentIDStr := c.FormValue("department")

	// Validasi
	if title == "" || description == "" || replyToEmail == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Semua field wajib diisi")
	}

	// Parse department ID
	var departmentID *uint
	if departmentIDStr != "" {
		id, err := strconv.ParseUint(departmentIDStr, 10, 32)
		if err == nil {
			uid := uint(id)
			departmentID = &uid
		}
	}

	// Create ticket
	ticket := models.Ticket{
		Title:        title,
		Description:  description,
		ReplyToEmail: replyToEmail,
		Priority:     models.TicketPriority(priority),
		Status:       models.StatusWaiting,
		CreatedByID:  user.ID,
		DepartmentID: departmentID,
	}

	if err := config.DB.Create(&ticket).Error; err != nil {
		log.Printf("Failed to create ticket: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to create ticket")
	}

	// Load relations untuk email
	config.DB.Preload("Department").First(&ticket, ticket.ID)

	// Send confirmation email
	departmentName := "Tidak Ditentukan"
	if ticket.Department != nil {
		departmentName = ticket.Department.Name
	}

	err := h.emailService.SendTicketConfirmation(
		replyToEmail,
		user.GetFullName(),
		ticket.Title,
		ticket.ID,
		departmentName,
		ticket.GetPriorityDisplay(),
		ticket.GetStatusDisplay(),
		ticket.Description,
	)

	if err != nil {
		log.Printf("Failed to send confirmation email: %v", err)
		// Continue anyway, ticket sudah dibuat
	}

	log.Printf("Ticket #%d created by user %s", ticket.ID, user.Username)

	return c.Redirect(fmt.Sprintf("/tiket/sukses/%d", ticket.ID))
}

// ShowTicketSuccess menampilkan halaman sukses
func (h *TicketHandler) ShowTicketSuccess(c *fiber.Ctx) error {
	ticketID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Redirect("/dashboard")
	}

	var ticket models.Ticket
	if err := config.DB.First(&ticket, ticketID).Error; err != nil {
		return c.Redirect("/dashboard")
	}

	return c.Render("tickets/ticket_success", fiber.Map{
		"title":  "Tiket Berhasil Dibuat",
		"ticket": ticket,
	})
}

// ShowMyTickets menampilkan daftar tiket user
func (h *TicketHandler) ShowMyTickets(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	// Get filter parameters
	searchQuery := c.Query("search", "")
	statusFilter := c.Query("status", "all")
	priorityFilter := c.Query("priority", "all")

	// Build query
	query := config.DB.Preload("Department").
		Preload("Replies").
		Where("created_by_id = ?", user.ID)

	// Apply search filter
	if searchQuery != "" {
		// Try to parse as ticket ID
		if ticketID, err := strconv.Atoi(searchQuery); err == nil {
			query = query.Where("id = ? OR title LIKE ? OR description LIKE ?",
				ticketID,
				"%"+searchQuery+"%",
				"%"+searchQuery+"%")
		} else {
			query = query.Where("title LIKE ? OR description LIKE ?",
				"%"+searchQuery+"%",
				"%"+searchQuery+"%")
		}
	}

	// Apply status filter
	if statusFilter != "all" {
		var status models.TicketStatus
		switch statusFilter {
		case "open":
			status = models.StatusWaiting
		case "in_progress":
			status = models.StatusInProgress
		case "closed":
			status = models.StatusClosed
		}
		query = query.Where("status = ?", status)
	}

	// Apply priority filter
	if priorityFilter != "all" {
		query = query.Where("priority = ?", priorityFilter)
	}

	// Get tickets
	var tickets []models.Ticket
	query.Order("created_at DESC").Find(&tickets)

	return c.Render("tickets/my_tickets", fiber.Map{
		"title":           "Tiket Saya - Portal Ticketing",
		"tickets":         tickets,
		"search_query":    searchQuery,
		"status_filter":   statusFilter,
		"priority_filter": priorityFilter,
	})
}

// ShowTicketDetail menampilkan detail tiket
func (h *TicketHandler) ShowTicketDetail(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	ticketID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Redirect("/tiket")
	}

	// Get ticket with all relations
	var ticket models.Ticket
	if err := config.DB.Preload("CreatedBy").
		Preload("Department").
		Preload("Replies.User").
		Where("id = ? AND created_by_id = ?", ticketID, user.ID).
		First(&ticket).Error; err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Ticket not found")
	}

	return c.Render("tickets/ticket_detail", fiber.Map{
		"title":   fmt.Sprintf("Tiket #%d - %s", ticket.ID, ticket.Title),
		"ticket":  ticket,
		"replies": ticket.Replies,
	})
}

// AddReply menambahkan reply ke tiket
func (h *TicketHandler) AddReply(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	ticketID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Redirect("/tiket")
	}

	message := c.FormValue("message")
	if message == "" {
		return c.Redirect(fmt.Sprintf("/tiket/%d", ticketID))
	}

	// Get ticket
	var ticket models.Ticket
	if err := config.DB.Preload("CreatedBy").
		Where("id = ? AND created_by_id = ?", ticketID, user.ID).
		First(&ticket).Error; err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Ticket not found")
	}

	// Create reply
	reply := models.TicketReply{
		TicketID: ticket.ID,
		UserID:   user.ID,
		Message:  message,
	}

	if err := config.DB.Create(&reply).Error; err != nil {
		log.Printf("Failed to create reply: %v", err)
		return c.Redirect(fmt.Sprintf("/tiket/%d", ticketID))
	}

	// Update ticket updated_at
	config.DB.Model(&ticket).Update("updated_at", "NOW()")

	log.Printf("Reply added to ticket #%d by user %s", ticketID, user.Username)

	return c.Redirect(fmt.Sprintf("/tiket/%d", ticketID))
}
