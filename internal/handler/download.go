package handler

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/SyNdicateFoundation/SteamDownloaderAPI/internal/steam"
	"github.com/SyNdicateFoundation/SteamDownloaderAPI/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/schollz/progressbar/v3"
)

func (h *SteamDownloaderAPI) DownloadWorkshopHandler(c *gin.Context) {
	appID, err := strconv.Atoi(c.Param("app_id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid App ID.")
		return
	}

	workshopID, err := strconv.Atoi(c.Param("workshop_id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid Workshop ID.")
		return
	}

	workshopName, err := steam.GetWorkshopName(workshopID)
	if err != nil {
		c.String(http.StatusNotFound, "Could not find workshop item: %v", err)
		return
	}

	zipFileName := fmt.Sprintf("%d_%s.zip", workshopID, util.SanitizeFileName(workshopName))
	zipFilePath := filepath.Join(h.saveDirectory, zipFileName)

	if _, err := os.Stat(zipFilePath); !os.IsNotExist(err) {
		c.FileAttachment(zipFilePath, zipFileName)
		return
	}

	log.Printf("⬇️ Starting download for AppID: %d, WorkshopID: %d", appID, workshopID)

	if err := h.steamcmd.DownloadWorkshopItem(appID, workshopID, true); err != nil {
		c.String(http.StatusInternalServerError, "Failed to download item: %v", err)
		return
	}
	log.Printf("✅ Downloaded AppID: %d, WorkshopID: %d. Now zipping...", appID, workshopID)

	sourcePath := h.steamcmd.GetWorkshopContentPath(appID, workshopID)

	if err := util.ZipDirectory(sourcePath, zipFilePath); err != nil {
		c.String(http.StatusInternalServerError, "Failed to create zip archive: %v", err)
		return
	}
	log.Printf("📦 Zipped successfully: %s", zipFileName)

	c.FileAttachment(zipFilePath, zipFileName)
}

func (h *SteamDownloaderAPI) DownloadCollectionHandler(c *gin.Context) {
	appID, _ := strconv.Atoi(c.Param("app_id"))
	collectionID, _ := strconv.Atoi(c.Param("collection_id"))

	log.Printf("⬇️ Starting download for CollectionID: %d", collectionID)

	collectionTitle, items, err := steam.GetCollectionItems(collectionID)
	if err != nil {
		c.String(http.StatusNotFound, "Could not get collection items: %v", err)
		return
	}

	if len(items) == 0 {
		c.String(http.StatusNotFound, "Collection is empty or could not be found.")
		return
	}

	zipFileName := fmt.Sprintf("%d_%s_collection.zip", collectionID, util.SanitizeFileName(collectionTitle))
	zipFilePath := filepath.Join(h.saveDirectory, zipFileName)

	if _, err := os.Stat(zipFilePath); !os.IsNotExist(err) {
		c.FileAttachment(zipFilePath, zipFileName)
		return
	}

	log.Printf("Collection '%s' contains %d items.", collectionTitle, len(items))

	bar := progressbar.Default(
		int64(len(items)),
		"Downloading collection items",
	)

	const maxWorkers = 5
	var wg sync.WaitGroup
	itemChan := make(chan steam.WorkshopItem, len(items))

	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range itemChan {
				if err := h.steamcmd.DownloadWorkshopItem(appID, item.ID, false); err != nil {
					log.Printf("   ⚠️ Failed to download item %d (%s): %v\n", item.ID, item.Title, err)
					continue
				}
				bar.Add(1)
			}
		}()
	}

	for _, item := range items {
		itemChan <- item
	}

	close(itemChan)
	wg.Wait()

	log.Println("✅ All collection items downloaded. Now zipping...")

	var contentPaths []util.ZipSource
	for _, item := range items {
		contentPaths = append(contentPaths, util.ZipSource{
			Path:  h.steamcmd.GetWorkshopContentPath(appID, item.ID),
			Alias: fmt.Sprintf("%d_%s", item.ID, util.SanitizeFileName(item.Title)),
		})
	}

	if err := util.ZipMultipleDirectories(contentPaths, zipFilePath); err != nil {
		c.String(http.StatusInternalServerError, "Failed to create collection zip: %v", err)
		return
	}
	log.Printf("📦 Zipped collection successfully: %s", zipFileName)

	c.FileAttachment(zipFilePath, zipFileName)
}
