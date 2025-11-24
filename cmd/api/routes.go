package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	movieDelivery "github.com/martinmanurung/cinestream/internal/domain/movies/delivery"
	orderDelivery "github.com/martinmanurung/cinestream/internal/domain/orders/delivery"
	userDelivery "github.com/martinmanurung/cinestream/internal/domain/users/delivery"
	"github.com/martinmanurung/cinestream/pkg/jwt"
	appMiddleware "github.com/martinmanurung/cinestream/pkg/middleware"
	"github.com/martinmanurung/cinestream/pkg/response"
)

func setupRoutes(e *echo.Echo, userHandler *userDelivery.Handler, movieHandler *movieDelivery.MovieHandler, genreHandler *movieDelivery.GenreHandler, orderHandler *orderDelivery.OrderHandler, webhookHandler *orderDelivery.WebhookHandler, streamingHandler *orderDelivery.StreamingHandler, jwtService *jwt.JWTService) {
	// Middleware
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Gzip())
	e.Use(middleware.CORS())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())

	// Custom error handler
	e.HTTPErrorHandler = response.CustomErrorHandler

	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{
			"status": "ok",
		})
	})

	// API v1 routes
	v1 := e.Group("/api/v1")

	// User routes
	users := v1.Group("/users")
	{
		users.POST("/register", userHandler.RegisterUser)
		users.POST("/login", userHandler.LoginUser)
		users.POST("/logout", userHandler.Logout)
		users.POST("/refresh", userHandler.RefreshToken)

		// Protected routes (require JWT)
		users.GET("/me", userHandler.GetMe, jwtService.JWTMiddleware())
	}

	// Movie routes (Public)
	movies := v1.Group("/movies")
	{
		movies.GET("", movieHandler.GetMovieList)       // GET /api/v1/movies?page=1&limit=12&genre=action
		movies.GET("/:id", movieHandler.GetMovieDetail) // GET /api/v1/movies/:id
	}

	// Genre routes (Public)
	genres := v1.Group("/genres")
	{
		genres.GET("", genreHandler.GetAllGenres) // GET /api/v1/genres
	}

	// Order routes
	orders := v1.Group("/orders")
	{
		// Protected user routes (require JWT)
		orders.POST("", orderHandler.CreateOrder, jwtService.JWTMiddleware())                                 // POST /api/v1/orders (create rental order)
		orders.GET("/me", orderHandler.GetUserOrders, jwtService.JWTMiddleware())                             // GET /api/v1/orders/me (user's order history)
		orders.GET("/:id", orderHandler.GetOrderDetail, jwtService.JWTMiddleware())                           // GET /api/v1/orders/:id (order detail)
		orders.POST("/:id/simulate-payment", orderHandler.SimulatePaymentSuccess, jwtService.JWTMiddleware()) // POST /api/v1/orders/:id/simulate-payment (dev only)
	}

	// Streaming endpoint (Protected with JWT)
	v1.GET("/movies/:id/stream", streamingHandler.GetStreamURL, jwtService.JWTMiddleware()) // GET /api/v1/movies/:id/stream

	// Webhook routes (Public but validated via signature)
	webhooks := v1.Group("/webhooks")
	{
		webhooks.POST("/payment", webhookHandler.HandlePaymentWebhook) // POST /api/v1/webhooks/payment (Midtrans notification)
	}

	// Admin routes (Protected with JWT + AdminOnly middleware)
	admin := v1.Group("/admin")
	admin.Use(jwtService.JWTMiddleware(), appMiddleware.AdminOnly())
	{
		// Admin movie management
		adminMovies := admin.Group("/movies")
		{
			adminMovies.POST("", movieHandler.UploadMovie)       // POST /api/v1/admin/movies
			adminMovies.GET("", movieHandler.GetAllMoviesAdmin)  // GET /api/v1/admin/movies?page=1&status=PENDING
			adminMovies.PUT("/:id", movieHandler.UpdateMovie)    // PUT /api/v1/admin/movies/:id
			adminMovies.DELETE("/:id", movieHandler.DeleteMovie) // DELETE /api/v1/admin/movies/:id
		}

		// Admin genre management
		adminGenres := admin.Group("/genres")
		{
			adminGenres.POST("", genreHandler.CreateGenre)       // POST /api/v1/admin/genres
			adminGenres.DELETE("/:id", genreHandler.DeleteGenre) // DELETE /api/v1/admin/genres/:id
		}

		// Admin order management
		adminOrders := admin.Group("/orders")
		{
			adminOrders.GET("", orderHandler.GetAllOrders) // GET /api/v1/admin/orders?page=1&status=PAID
		}
	}

	// orders := v1.Group("/orders")
}
