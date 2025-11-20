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
		CookieSecure:   false, // Set true in production with HTTPS
		CookieSameSite: "Lax",
	})

	// Initialize template engine
	engine := html.New("./templates", ".html")
	engine.Reload(cfg.Debug)
	engine.Debug(cfg.Debug)

	// Add custom template functions
	engine.AddFunc("slice", func(s string, start, end int) string {
		if start < 0 || end > len(s) || start > end {
			return s
		}
		return s[start:end]
	})

	engine.AddFunc("upper", func(s string) string {
		return s
	})

	engine.AddFunc("default", func(value, defaultValue interface{}) interface{} {
		if value == nil || value == "" {
			return defaultValue
		}
		return value
	})

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

	engine.AddFunc("eq", func(a, b interface{}) bool {
		return a == b
	})

	engine.AddFunc("len", func(arr interface{}) int {
		switch v := arr.(type) {
		case []interface{}:
			return len(v)
		case []models.Ticket:
			return len(v)
		case []models.TicketReply:
			return len(v)
		case []models.Department:
			return len(v)
		}
		return 0
	})

	engine.AddFunc("linebreaks", func(s string) string {
		// Convert newlines to <br> tags
		return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "<br>"), "\n", "<br>")
	})

	engine.AddFunc("getFullName", func(user interface{}) string {
		if user == nil {
			return ""
		}
		if u, ok := user.(*models.User); ok {
			if u.FirstName != "" || u.LastName != "" {
				return strings.TrimSpace(u.FirstName + " " + u.LastName)
			}
			return u.Username
		}
		return ""
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

	// Set user locals for all routes
	app.Use(middleware.SetUserLocals)

	// Initialize services
	emailService := utils.NewEmailService(cfg)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(cfg)
	dashboardHandler := handlers.NewDashboardHandler(cfg)
	ticketHandler := handlers.NewTicketHandler(cfg, emailService)
	settingsHandler := handlers.NewSettingsHandler(cfg)

	// Routes - Public
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/login")
	})

	// Auth routes
	app.Get("/login", middleware.GuestOnly, authHandler.ShowLogin)
	app.Post("/login", authHandler.Login)
	app.Get("/register", middleware.GuestOnly, authHandler.ShowRegister)
	app.Post("/register", authHandler.Register)
	app.Get("/logout", authHandler.Logout)
	app.Post("/logout", authHandler.Logout)

	// Protected routes
	protected := app.Group("/", middleware.AuthRequired, middleware.PortalUserRequired)

	// Dashboard
	protected.Get("/dashboard", dashboardHandler.ShowDashboard)

	// Tickets
	protected.Get("/tiket", ticketHandler.ShowMyTickets)
	protected.Get("/tiket/:id", ticketHandler.ShowTicketDetail)
	protected.Post("/tiket/:id", ticketHandler.AddReply)
	protected.Get("/kirim-tiket", ticketHandler.ShowCreateTicket)
	protected.Post("/kirim-tiket", ticketHandler.CreateTicket)
	protected.Get("/tiket/sukses/:id", ticketHandler.ShowTicketSuccess)

	// Settings
	protected.Get("/settings", settingsHandler.ShowSettings)
	protected.Post("/settings/profile", settingsHandler.UpdateProfile)
	protected.Post("/settings/password", settingsHandler.ChangePassword)

	// Seed default data (run once)
	seedDefaultData()

	// Start server
	log.Printf("🚀 Server starting on port %s", cfg.Port)
	log.Printf("🌐 Visit: http://localhost:%s", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}

func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	// Log error detail
	log.Printf("Error: %v", err)
	log.Printf("Error Type: %T", err)
	if code == fiber.StatusInternalServerError {
		log.Printf("Stack trace: %+v", err)
	}

	if code == fiber.StatusNotFound {
		return c.Status(code).SendString("Page not found")
	}

	// Return detailed error in debug mode
	return c.Status(code).SendString(fmt.Sprintf("Internal Server Error: %v", err))
}

func seedDefaultData() {
	// Create Portal Users group if not exists
	var portalGroup models.Group
	config.DB.FirstOrCreate(&portalGroup, models.Group{Name: "Portal Users"})

	// Create default departments if not exist
	departments := []string{"Technical Support", "Customer Service", "Billing", "General"}
	for _, deptName := range departments {
		var dept models.Department
		config.DB.FirstOrCreate(&dept, models.Department{Name: deptName})
	}

	// Create admin user if not exists (optional)
	var adminUser models.User
	if err := config.DB.Where("username = ?", "admin").First(&adminUser).Error; err != nil {
		hashedPassword, _ := utils.HashPassword("admin123")
		adminUser = models.User{
			Username:  "admin",
			Email:     "admin@ticketing.local",
			Password:  hashedPassword,
			FirstName: "Admin",
			LastName:  "System",
			IsStaff:   true,
			IsActive:  true,
		}
		config.DB.Create(&adminUser)
		config.DB.Model(&adminUser).Association("Groups").Append(&portalGroup)
		log.Println("✅ Admin user created - username: admin, password: admin123")
	}

	log.Println("✅ Default data seeded")
}
