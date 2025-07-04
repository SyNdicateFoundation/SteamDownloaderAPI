package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/SyNdicateFoundation/SteamDownloaderAPI/internal/handler"
	"github.com/SyNdicateFoundation/SteamDownloaderAPI/internal/steamcmd"
	"github.com/gin-gonic/gin"
)

func main() {
	steamCmdPath := flag.String("steamcmdpath", "steamcmd", "Path to the steamcmd directory")
	port := flag.String("port", "8080", "Port for the server to listen on")
	install := flag.Bool("install", true, "Install or update steamcmd on startup")
	flag.Parse()

	s, err := steamcmd.New(*steamCmdPath)
	if err != nil {
		log.Fatalf("❌ SteamCMD initialization error: %v", err)
	}

	if *install {
		if err := os.MkdirAll(*steamCmdPath, 0755); err != nil {
			log.Fatalf("❌ Failed to create steamcmd directory: %v", err)
		}

		if err := s.Install(); err != nil {
			log.Printf("⚠️ SteamCMD installation warning: %v", err)
		}

		router := gin.Default()

		h := handler.New(s)
		defer h.Cleanup()

		router.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, "/workshop/")
		})

		router.GET("/api/workshop/:app_id/:workshop_id", h.DownloadWorkshopHandler)
		router.GET("/api/collection/:app_id/:collection_id", h.DownloadCollectionHandler)

		router.Any("/workshop/*path", h.SteamProxyHandler)
		router.Any("/app/*path", h.SteamProxyHandler)
		router.Any("/sharedfiles/*path", h.SteamProxyHandler)

		unsupportedRoutes := []string{
			"/login/home/", "/market/", "/discussions/", "/my/",
			"/id/", "/account/", "/profiles/",
		}
		for _, route := range unsupportedRoutes {
			router.GET(route, h.UnsupportedPageHandler)
		}

		log.Printf("🚀 Server starting on http://0.0.0.0:%s", *port)
		if err := router.Run(":" + *port); err != nil {
			log.Fatalf("❌ Failed to start server: %v", err)
		}
	}
}
