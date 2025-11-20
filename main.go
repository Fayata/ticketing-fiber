// main.go
package main

import (
	"fmt"
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
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize database
	if err := config.InitDatabase(cfg); err != nil {
		log.Fatal(err)
	}

	// Auto migrate models
	if err := config.AutoMigrate(
		&models.User{},
		&models.Group{},
		&models.Department{},
		&models.Ticket{},
		&models.TicketReply{},
	); err != nil {
		log.Fatal(err)
	}

	// Initialize session store
	config.Store = session.New(session.Config{
		Expiration:     cfg.SessionExpiry,
		CookieHTTPOnly: true,
		CookieSecure:   false, // Set true in production
		CookieSameSite: "Lax",
	})

	// Initialize template engine
	engine := html.New("./templates", ".html")
	engine.Reload(cfg.Debug)
	engine.Debug(cfg.Debug)

	// --- TEMPLATE FUNCTIONS (HELPER) ---
	// Fungsi-fungsi ini meniru filter template Django

	// 1. Slice string (untuk avatar)
	engine.AddFunc("slice", func(s string, start, end int) string {
		if start < 0 || end > len(s) || start > end {
			return s
		}
		return s[start:end]
	})

	// 2. Uppercase
	engine.AddFunc("upper", func(s string) string {
		return strings.ToUpper(s)
	})

	// 3. Date formatting standar
	engine.AddFunc("date", func(t interface{}) string {
		if t == nil {
			return ""
		}
		switch v := t.(type) {
		case time.Time:
			return v.Format("02 Jan 2006, 15:04")
		case *time.Time:
			if v == nil {
				return ""
			}
			return v.Format("02 Jan 2006, 15:04")
		}
		return ""
	})

	// 4. Date formatting pendek
	engine.AddFunc("dateShort", func(t interface{}) string {
		if t == nil {
			return ""
		}
		switch v := t.(type) {
		case time.Time:
			return v.Format("02 Jan 2006")
		case *time.Time:
			if v == nil {
				return ""
			}
			return v.Format("02 Jan 2006")
		}
		return ""
	})

	// 5. Time Since (Meniru filter |timesince Django) - SOLUSI ERROR ANDA
	engine.AddFunc("timeSince", func(t time.Time) string {
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

	// 6. Equality check
	engine.AddFunc("eq", func(a, b interface{}) bool {
		return a == b
	})

	// 7. Length check
	engine.AddFunc("len", func(arr interface{}) int {
		if arr == nil {
			return 0
		}
		switch v := arr.(type) {
		case []interface{}:
			return len(v)
		case []models.Ticket:
			return len(v)
		case []models.TicketReply:
			return len(v)
		case []models.Department:
			return len(v)
		case string:
			return len(v)
		}
		return 0
	})

	// 8. Linebreaks (dengan perbaikan tipe data interface{})
	engine.AddFunc("linebreaks", func(val interface{}) string {
		var s string
		if val == nil {
			return ""
		}
		s = fmt.Sprint(val)
		return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "<br>"), "\n", "<br>")
	})

	// 9. Get Full Name (Menangani pointer dan value user)
	engine.AddFunc("getFullName", func(user interface{}) string {
		if user == nil {
			return "User"
		}
		// Handle jika user adalah pointer (*models.User)
		if u, ok := user.(*models.User); ok {
			if u.FirstName != "" || u.LastName != "" {
				return strings.TrimSpace(u.FirstName + " " + u.LastName)
			}
			return u.Username
		}
		// Handle jika user adalah value (models.User)
		if u, ok := user.(models.User); ok {
			if u.FirstName != "" || u.LastName != "" {
				return strings.TrimSpace(u.FirstName + " " + u.LastName)
			}
			return u.Username
		}
		return "User"
	})

	// 10. Truncate words (untuk deskripsi pengumuman)
	engine.AddFunc("truncatewords", func(s string, limit int) string {
		words := strings.Fields(s)
		if len(words) <= limit {
			return s
		}
		return strings.Join(words[:limit], " ") + "..."
	})

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		Views:        engine,
		ErrorHandler: customErrorHandler,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format:     "${time} | ${status} | ${latency} | ${method} ${path}\n",
		TimeFormat: "15:04:05",
	}))

	// Static files
	app.Static("/static", "./static")

	// Set user locals
	app.Use(middleware.SetUserLocals)

	// Services & Handlers
	emailService := utils.NewEmailService(cfg)
	authHandler := handlers.NewAuthHandler(cfg)
	dashboardHandler := handlers.NewDashboardHandler(cfg)
	ticketHandler := handlers.NewTicketHandler(cfg, emailService)
	settingsHandler := handlers.NewSettingsHandler(cfg)

	// Routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/login")
	})

	// Auth
	app.Get("/login", middleware.GuestOnly, authHandler.ShowLogin)
	app.Post("/login", authHandler.Login)
	app.Get("/register", middleware.GuestOnly, authHandler.ShowRegister)
	app.Post("/register", authHandler.Register)
	app.Get("/logout", authHandler.Logout)
	app.Post("/logout", authHandler.Logout)

	// Protected Routes
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

	// Seed Data
	seedDefaultData()

	// Start Server
	log.Printf("🚀 Server starting on port %s", cfg.Port)
	log.Printf("🌐 Visit: http://localhost:%s", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}

func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}
	log.Printf("Error: %v", err)
	// Return detailed error for debugging
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
