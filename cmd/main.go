package main

import (
	"flag"
	"log"
	"net"
	"net/http"

	_ "embed"
	"os"

	"github.com/SyNdicateFoundation/SteamDownloaderAPI/internal/handler"
	"github.com/SyNdicateFoundation/SteamDownloaderAPI/internal/steamcmd"
	"github.com/gin-gonic/gin"
)

var (
	steamCmdPath, listenHost, listenPort, steamUser, steamPassword string
	installSteamCmd, debugMode                                     bool
)

func init() {
	flag.StringVar(&steamCmdPath, "steamcmdpath", "steamcmd", "Path to the steamcmd directory")
	flag.BoolVar(&installSteamCmd, "installsteamcmd", true, "Install steamcmd")
	flag.BoolVar(&debugMode, "debug", false, "Install steamcmd")
	flag.StringVar(&listenHost, "listenhost", "0.0.0.0", "Hostname for the server to listen on")
	flag.StringVar(&listenPort, "listenport", "8080", "Port for the server to listen on")
	flag.StringVar(&steamUser, "steamuser", "", "Steam username")
	flag.StringVar(&steamPassword, "steampassword", "", "Steam password")

	flag.Parse()
}

//go:embed favicon.ico
var favicon []byte

func main() {
	s, err := steamcmd.New(steamCmdPath, steamUser, steamPassword)
	if err != nil {
		log.Fatalf("❌ SteamCMD initialization error: %v", err)
	}

	if installSteamCmd {
		if err := os.MkdirAll(steamCmdPath, 0755); err != nil {
			log.Fatalf("❌ Failed to create steamcmd directory: %v", err)
		}

		if err := s.Install(); err != nil {
			log.Printf("⚠️ SteamCMD installation warning: %v", err)
		}
	}

	gin.SetMode(gin.ReleaseMode)

	if debugMode {
		gin.SetMode(gin.DebugMode)
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
	router.Any("/public/*path", h.SteamProxyHandler)
	router.Any("/sharedfiles/*path", h.SteamProxyHandler)

	router.GET("favicon.ico", func(c *gin.Context) {
		c.Data(http.StatusOK, "image/x-icon", favicon)
	})

	unsupportedRoutes := []string{
		"/login/home/", "/market/", "/discussions/", "/my/",
		"/id/", "/account/", "/profiles/",
	}
	for _, route := range unsupportedRoutes {
		router.GET(route, h.UnsupportedPageHandler)
	}

	listenAddr := net.JoinHostPort(listenHost, listenPort)

	log.Printf("🚀 Server starting on http://%s", listenAddr)

	if err := router.Run(listenAddr); err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
}
