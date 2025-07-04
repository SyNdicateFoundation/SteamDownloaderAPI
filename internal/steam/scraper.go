package steam

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type WorkshopItem struct {
	ID    int
	Title string
}

func GetWorkshopName(workshopID int) (string, error) {
	url := fmt.Sprintf("https://steamcommunity.com/sharedfiles/filedetails/?id=%d", workshopID)
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return "", fmt.Errorf("steam returned status %d", res.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", err
	}

	title := doc.Find("div.workshopItemTitle").First().Text()
	if title == "" {
		return "", fmt.Errorf("could not find title for workshop item %d", workshopID)
	}
	return strings.TrimSpace(title), nil
}

func GetCollectionItems(collectionID int) (string, []WorkshopItem, error) {
	url := fmt.Sprintf("https://steamcommunity.com/sharedfiles/filedetails/?id=%d", collectionID)
	res, err := http.Get(url)
	if err != nil {
		return "", nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return "", nil, fmt.Errorf("steam returned status %d", res.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", nil, err
	}

	collectionTitle := doc.Find("div.workshopItemTitle").First().Text()
	var items []WorkshopItem

	doc.Find("div.collectionItem").Each(func(i int, s *goquery.Selection) {
		title := s.Find("div.collectionItemTitle").Text()
		idStr, exists := s.Find("a").Attr("href")
		if !exists {
			return
		}
		idParts := strings.Split(idStr, "=")
		if len(idParts) != 2 {
			return
		}
		id, err := strconv.Atoi(idParts[1])
		if err != nil {
			return
		}
		items = append(items, WorkshopItem{ID: id, Title: strings.TrimSpace(title)})
	})

	return strings.TrimSpace(collectionTitle), items, nil
}
