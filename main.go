// File: main.go
package main

import (
	"fmt"
	"html/template"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/template/html/v2"

	"ticketing-fiber/config"
	"ticketing-fiber/handlers"
	"ticketing-fiber/middleware"
	"ticketing-fiber/models"
	"ticketing-fiber/utils"
)

func main() {
	cfg := config.LoadConfig()

	if err := config.InitDatabase(cfg); err != nil {
		log.Fatal(err)
	}

	if err := config.AutoMigrate(&models.User{}, &models.Group{}, &models.Department{}, &models.Ticket{}, &models.TicketReply{}); err != nil {
		log.Fatal(err)
	}

	config.Store = session.New(session.Config{
		Expiration:     cfg.SessionExpiry,
		CookieHTTPOnly: true,
		CookieSecure:   false,
		CookieSameSite: "Lax",
	})

	engine := html.New("./templates", ".html")
	engine.Reload(cfg.Debug)
	engine.Debug(cfg.Debug)

	// --- TEMPLATE FUNCTIONS (PERBAIKAN UTAMA) ---

	// 1. Helper untuk Waktu (Persis Django |timesince)
	engine.AddFunc("timeSince", func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		now := time.Now()
		diff := now.Sub(t)

		days := int(diff.Hours() / 24)
		hours := int(diff.Hours())
		minutes := int(diff.Minutes())

		if days > 0 {
			return fmt.Sprintf("%d hari", days)
		}
		if hours > 0 {
			return fmt.Sprintf("%d jam", hours)
		}
		if minutes > 0 {
			return fmt.Sprintf("%d menit", minutes)
		}
		return "Baru saja"
	})

	// 2. Helper untuk CSS Class Status (Agar tampilan tidak berantakan)
	// Menangani status lama (OPEN) dan baru (WAITING)
	engine.AddFunc("getStatusClass", func(status string) string {
		switch status {
		case "WAITING", "OPEN":
			return "open"
		case "IN_PROGRESS":
			return "in-progress"
		case "CLOSED", "RESOLVED":
			return "closed"
		default:
			return "closed"
		}
	})

	// 3. Helper untuk CSS Class Priority
	engine.AddFunc("getPriorityClass", func(priority string) string {
		switch priority {
		case "HIGH":
			return "high"
		case "MEDIUM":
			return "medium"
		case "LOW":
			return "low"
		default:
			return "low"
		}
	})

	// 4. Helper umum lainnya
	engine.AddFunc("slice", func(s string, start, end int) string {
		if start < 0 || end > len(s) || start > end {
			return s
		}
		return s[start:end]
	})

	engine.AddFunc("upper", func(s string) string { return strings.ToUpper(s) })

	engine.AddFunc("date", func(t time.Time) string {
		return t.Format("02 Jan 2006, 15:04")
	})

	engine.AddFunc("dateShort", func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.Format("02 Jan 2006")
	})

	engine.AddFunc("linebreaks", func(val interface{}) template.HTML {
		if val == nil {
			return ""
		}
		s := template.HTMLEscapeString(fmt.Sprint(val))
		s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "<br>"), "\n", "<br>")
		return template.HTML(s)
	})

	engine.AddFunc("getFullName", func(user interface{}) string {
		if user == nil {
			return "User"
		}
		if u, ok := user.(*models.User); ok {
			if u.FirstName != "" {
				return strings.TrimSpace(u.FirstName + " " + u.LastName)
			}
			return u.Username
		}
		if u, ok := user.(models.User); ok {
			if u.FirstName != "" {
				return strings.TrimSpace(u.FirstName + " " + u.LastName)
			}
			return u.Username
		}
		return "User"
	})

	// Init App
	app := fiber.New(fiber.Config{
		Views:        engine,
		ErrorHandler: customErrorHandler,
	})

	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format:     "${time} | ${status} | ${latency} | ${method} ${path}\n",
		TimeFormat: "15:04:05",
	}))

	// Static Files
	app.Static("/static", "./static")

	// Middleware
	app.Use(middleware.SetUserLocals)

	// Services
	emailService := utils.NewEmailService(cfg)

	// Handlers
	authHandler := handlers.NewAuthHandler(cfg)
	dashboardHandler := handlers.NewDashboardHandler(cfg)
	ticketHandler := handlers.NewTicketHandler(cfg, emailService)
	settingsHandler := handlers.NewSettingsHandler(cfg)

	// Routes
	app.Get("/", func(c *fiber.Ctx) error { return c.Redirect("/login") })
	app.Get("/login", middleware.GuestOnly, authHandler.ShowLogin)
	app.Post("/login", authHandler.Login)
	app.Get("/register", middleware.GuestOnly, authHandler.ShowRegister)
	app.Post("/register", authHandler.Register)
	app.Get("/logout", authHandler.Logout)
	app.Post("/logout", authHandler.Logout)

	protected := app.Group("/", middleware.AuthRequired, middleware.PortalUserRequired)
	protected.Get("/dashboard", dashboardHandler.ShowDashboard)
	protected.Get("/tiket", ticketHandler.ShowMyTickets)
	protected.Get("/tiket/:id", ticketHandler.ShowTicketDetail)
	protected.Post("/tiket/:id", ticketHandler.AddReply)
	protected.Get("/kirim-tiket", ticketHandler.ShowCreateTicket)
	protected.Post("/kirim-tiket", ticketHandler.CreateTicket)
	protected.Get("/tiket/sukses/:id", ticketHandler.ShowTicketSuccess)
	protected.Get("/settings", settingsHandler.ShowSettings)
	protected.Post("/settings/profile", settingsHandler.UpdateProfile)
	protected.Post("/settings/password", settingsHandler.ChangePassword)

	seedDefaultData()

	log.Printf("🚀 Server starting on port %s", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}

func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}
	log.Printf("Error: %v", err)
	return c.Status(code).SendString(fmt.Sprintf("Internal Server Error: %v", err))
}

func seedDefaultData() {
	var portalGroup models.Group
	config.DB.FirstOrCreate(&portalGroup, models.Group{Name: "Portal Users"})
	departments := []string{"Technical Support", "Customer Service", "Billing", "General"}
	for _, deptName := range departments {
		var dept models.Department
		config.DB.FirstOrCreate(&dept, models.Department{Name: deptName})
	}
}
