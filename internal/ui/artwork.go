package ui

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/gen2brain/webp"

	"github.com/lobinuxsoft/bazzite-devkit/internal/config"
)

// tappableImage is a custom widget that shows an image and can be tapped
type tappableImage struct {
	widget.BaseWidget
	content   fyne.CanvasObject
	onTap     func()
	hovered   bool
	border    *canvas.Rectangle
}

func newTappableImage(content fyne.CanvasObject, border *canvas.Rectangle, onTap func()) *tappableImage {
	t := &tappableImage{
		content: content,
		onTap:   onTap,
		border:  border,
	}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tappableImage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.content)
}

func (t *tappableImage) Tapped(_ *fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

func (t *tappableImage) TappedSecondary(_ *fyne.PointEvent) {}

func (t *tappableImage) MouseIn(_ *desktop.MouseEvent) {
	t.hovered = true
	if t.border != nil {
		t.border.StrokeWidth = 2
		t.border.StrokeColor = color.RGBA{100, 100, 255, 200}
		t.border.Refresh()
	}
}

func (t *tappableImage) MouseMoved(_ *desktop.MouseEvent) {}

func (t *tappableImage) MouseOut() {
	t.hovered = false
	if t.border != nil {
		// Only reset if not selected (green)
		if t.border.StrokeColor != (color.RGBA{0, 255, 0, 255}) {
			t.border.StrokeWidth = 0
			t.border.StrokeColor = color.Transparent
			t.border.Refresh()
		}
	}
}

const steamGridDBBaseURL = "https://www.steamgriddb.com/api/v2"

// ArtworkSelection holds the selected artwork URLs
type ArtworkSelection struct {
	GridDBGameID  int
	GridPortrait  string
	GridLandscape string
	HeroImage     string
	LogoImage     string
	IconImage     string
}

// API Response types
type sgdbResponse struct {
	Success bool     `json:"success"`
	Errors  []string `json:"errors"`
}

type sgdbSearchResponse struct {
	sgdbResponse
	Data []sgdbSearchResult `json:"data"`
}

type sgdbSearchResult struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Types    []string `json:"types"`
	Verified bool     `json:"verified"`
}

type sgdbGridResponse struct {
	sgdbResponse
	Data []sgdbGridData `json:"data"`
}

type sgdbGridData struct {
	ID        int    `json:"id"`
	Score     int    `json:"score"`
	Style     string `json:"style"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Nsfw      bool   `json:"nsfw"`
	Humor     bool   `json:"humor"`
	Mime      string `json:"mime"`
	Language  string `json:"language"`
	URL       string `json:"url"`
	Thumb     string `json:"thumb"`
	Lock      bool   `json:"lock"`
	Epilepsy  bool   `json:"epilepsy"`
	Upvotes   int    `json:"upvotes"`
	Downvotes int    `json:"downvotes"`
}

type sgdbImageResponse struct {
	sgdbResponse
	Data []sgdbImageData `json:"data"`
}

type sgdbImageData struct {
	ID        int    `json:"id"`
	Score     int    `json:"score"`
	Style     string `json:"style"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Nsfw      bool   `json:"nsfw"`
	Humor     bool   `json:"humor"`
	Mime      string `json:"mime"`
	Language  string `json:"language"`
	URL       string `json:"url"`
	Thumb     string `json:"thumb"`
	Lock      bool   `json:"lock"`
	Epilepsy  bool   `json:"epilepsy"`
	Upvotes   int    `json:"upvotes"`
	Downvotes int    `json:"downvotes"`
}

// imageCache for in-memory caching
var imageCache = make(map[string]image.Image)
var imageCacheMu sync.RWMutex

// Filter options
type imageFilters struct {
	style     string
	mimeType  string
	imageType string // "static", "animated", or "" for all
	dimension string
	showNsfw  bool
	showHumor bool
}

// Asset type enum
type assetType int

const (
	assetCapsule assetType = iota
	assetWideCapsule
	assetHero
	assetLogo
	assetIcon
)

// Style options per asset type (from Decky plugin)
var gridStyles = []string{"All Styles", "alternate", "white_logo", "no_logo", "blurred", "material"}
var heroStyles = []string{"All Styles", "alternate", "blurred", "material"}
var logoStyles = []string{"All Styles", "official", "white", "black", "custom"}
var iconStyles = []string{"All Styles", "official", "custom"}

// Dimension options per asset type (from Decky plugin)
var capsuleDimensions = []string{"All Sizes", "600x900", "342x482", "660x930", "512x512", "1024x1024"}
var wideCapsuleDimensions = []string{"All Sizes", "460x215", "920x430"}
var heroDimensions = []string{"All Sizes", "1920x620", "3840x1240", "1600x650"}
var logoDimensions = []string{"All Sizes"}
var iconDimensions = []string{"All Sizes", "512x512", "256x256", "128x128", "64x64", "32x32", "24x24", "16x16"}

// MIME options per asset type
var gridMimes = []string{"All Formats", "image/png", "image/jpeg", "image/webp"}
var logoMimes = []string{"All Formats", "image/png", "image/webp"}
var iconMimes = []string{"All Formats", "image/png", "image/vnd.microsoft.icon"}

// Animation options
var animationOptions = []string{"All", "Static Only", "Animated Only"}

// SteamGridDB API client
type sgdbClient struct {
	apiKey string
	client http.Client
}

func newSGDBClient(apiKey string) *sgdbClient {
	return &sgdbClient{apiKey: apiKey}
}

func (c *sgdbClient) get(endpoint string, params url.Values) ([]byte, error) {
	reqURL := steamGridDBBaseURL + endpoint
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (c *sgdbClient) search(term string) ([]sgdbSearchResult, error) {
	body, err := c.get("/search/autocomplete/"+url.PathEscape(term), nil)
	if err != nil {
		return nil, err
	}

	var resp sgdbSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

func (c *sgdbClient) getGrids(gameID int, filters *imageFilters, page int) ([]sgdbGridData, error) {
	params := c.buildParams(filters, page)
	body, err := c.get(fmt.Sprintf("/grids/game/%d", gameID), params)
	if err != nil {
		return nil, err
	}

	var resp sgdbGridResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

func (c *sgdbClient) getHeroes(gameID int, filters *imageFilters, page int) ([]sgdbImageData, error) {
	params := c.buildParams(filters, page)
	body, err := c.get(fmt.Sprintf("/heroes/game/%d", gameID), params)
	if err != nil {
		return nil, err
	}

	var resp sgdbImageResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

func (c *sgdbClient) getLogos(gameID int, filters *imageFilters, page int) ([]sgdbImageData, error) {
	params := c.buildParams(filters, page)
	body, err := c.get(fmt.Sprintf("/logos/game/%d", gameID), params)
	if err != nil {
		return nil, err
	}

	var resp sgdbImageResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

func (c *sgdbClient) getIcons(gameID int, filters *imageFilters, page int) ([]sgdbImageData, error) {
	params := c.buildParams(filters, page)
	body, err := c.get(fmt.Sprintf("/icons/game/%d", gameID), params)
	if err != nil {
		return nil, err
	}

	var resp sgdbImageResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

func (c *sgdbClient) buildParams(filters *imageFilters, page int) url.Values {
	params := url.Values{}

	if filters.style != "" && filters.style != "All Styles" {
		params.Set("styles", filters.style)
	}
	if filters.mimeType != "" && filters.mimeType != "All Formats" {
		params.Set("mimes", filters.mimeType)
	}
	if filters.imageType == "static" {
		params.Set("types", "static")
	} else if filters.imageType == "animated" {
		params.Set("types", "animated")
	}
	// If empty, don't set types - API will return all
	if filters.dimension != "" && filters.dimension != "All Sizes" {
		params.Set("dimensions", filters.dimension)
	}
	if filters.showNsfw {
		params.Set("nsfw", "any")
	} else {
		params.Set("nsfw", "false")
	}
	if filters.showHumor {
		params.Set("humor", "any")
	} else {
		params.Set("humor", "false")
	}
	if page > 0 {
		params.Set("page", strconv.Itoa(page))
	}

	return params
}

// isAnimatedImage checks if an image is animated based on mime type and URL
func isAnimatedImage(mime, imgURL string) bool {
	mime = strings.ToLower(mime)
	imgURL = strings.ToLower(imgURL)

	// GIF is always potentially animated
	if mime == "image/gif" || strings.HasSuffix(imgURL, ".gif") {
		return true
	}

	// APNG is animated PNG
	if mime == "image/apng" {
		return true
	}

	// Check URL for animation indicators
	if strings.Contains(imgURL, "animated") || strings.Contains(imgURL, "anim") {
		return true
	}

	// Note: We don't assume all WebP are animated - only those with indicators
	return false
}

// ShowArtworkSelectionWindow shows the artwork selection window
func ShowArtworkSelectionWindow(gameName string, currentSelection *ArtworkSelection, onSave func(selection *ArtworkSelection)) {
	apiKey, err := config.GetSteamGridDBAPIKey()
	if err != nil || apiKey == "" {
		dialog.ShowError(fmt.Errorf("SteamGridDB API key not configured.\nPlease set it in Settings tab."), State.Window)
		return
	}

	artWindow := fyne.CurrentApp().NewWindow("Select Artwork - " + gameName)
	artWindow.Resize(fyne.NewSize(1100, 800))

	client := newSGDBClient(apiKey)
	selection := &ArtworkSelection{}
	if currentSelection != nil {
		*selection = *currentSelection
	}

	// Filters for each tab
	capsuleFilters := &imageFilters{showNsfw: false, showHumor: true}
	wideFilters := &imageFilters{showNsfw: false, showHumor: true}
	heroFilters := &imageFilters{showNsfw: false, showHumor: true}
	logoFilters := &imageFilters{showNsfw: false, showHumor: true}
	iconFilters := &imageFilters{showNsfw: false, showHumor: true}

	var searchResults []sgdbSearchResult
	var selectedGameID int

	// Preview image and label
	previewImage := canvas.NewImageFromImage(nil)
	previewImage.FillMode = canvas.ImageFillContain
	previewImage.SetMinSize(fyne.NewSize(250, 350))
	previewLabel := widget.NewLabel("Select an image to preview")
	previewLabel.Alignment = fyne.TextAlignCenter
	previewLabel.Wrapping = fyne.TextWrapWord

	// Current preview URL for "Open in Browser"
	var currentPreviewURL string

	// Open in browser button
	openBrowserBtn := widget.NewButtonWithIcon("Open in Browser", theme.ComputerIcon(), func() {
		if currentPreviewURL == "" {
			return
		}
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", currentPreviewURL)
		case "darwin":
			cmd = exec.Command("open", currentPreviewURL)
		default:
			cmd = exec.Command("xdg-open", currentPreviewURL)
		}
		cmd.Start()
	})
	openBrowserBtn.Importance = widget.MediumImportance

	// Function to update preview
	updatePreview := func(imgURL string, info string) {
		currentPreviewURL = imgURL
		previewLabel.SetText(info)
		go func() {
			img, err := downloadImage(imgURL)
			if err != nil {
				previewLabel.SetText(info + "\n\n(Failed to load: " + err.Error() + ")\nUse 'Open in Browser' to view")
				return
			}
			previewImage.Image = img
			previewImage.Refresh()
		}()
	}

	// Track selected thumbnails - pointer to the selected border rectangle
	var selectedCapsuleBorder, selectedWideBorder, selectedHeroBorder, selectedLogoBorder, selectedIconBorder *canvas.Rectangle

	statusLabel := widget.NewLabel("Search for a game to select artwork")

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search game name...")
	searchEntry.SetText(gameName)

	gameSelectLabel := widget.NewLabel("No game selected")

	// Containers for images
	capsuleContainer := container.NewGridWrap(fyne.NewSize(140, 240))
	wideContainer := container.NewGridWrap(fyne.NewSize(200, 120))
	heroContainer := container.NewGridWrap(fyne.NewSize(220, 100))
	logoContainer := container.NewGridWrap(fyne.NewSize(140, 160))
	iconContainer := container.NewGridWrap(fyne.NewSize(90, 110))

	// Selected labels
	selectedCapsuleLabel := widget.NewLabel("None selected")
	selectedWideLabel := widget.NewLabel("None selected")
	selectedHeroLabel := widget.NewLabel("None selected")
	selectedLogoLabel := widget.NewLabel("None selected")
	selectedIconLabel := widget.NewLabel("None selected")

	var capsulePage, widePage, heroPage, logoPage, iconPage int
	var loadMoreCapsule, loadMoreWide, loadMoreHero, loadMoreLogo, loadMoreIcon *widget.Button

	// Helper to create thumbnail with proper selection handling
	createThumb := func(thumbURL, fullURL, mime string, thumbSize fyne.Size, w, h int, style string, selectedBorder **canvas.Rectangle, selectedLabel *widget.Label, selectionTarget *string) fyne.CanvasObject {
		// Background
		bg := canvas.NewRectangle(color.RGBA{50, 50, 50, 255})

		// Selection/hover border - STARTS TRANSPARENT
		border := canvas.NewRectangle(color.Transparent)
		border.StrokeWidth = 0
		border.StrokeColor = color.Transparent

		// Loading indicator
		loadingLabel := widget.NewLabel("...")
		loadingLabel.Alignment = fyne.TextAlignCenter

		imgWidget := canvas.NewImageFromImage(nil)
		imgWidget.FillMode = canvas.ImageFillContain
		imgWidget.SetMinSize(thumbSize)

		// Container for image + loading
		imgStack := container.NewStack(bg, imgWidget, loadingLabel)

		// Load image async
		go func() {
			img, err := downloadImage(thumbURL)
			if err != nil {
				loadingLabel.SetText("!")
				return
			}
			imgWidget.Image = img
			imgWidget.Refresh()
			loadingLabel.SetText("")
		}()

		// Check if animated
		isAnim := isAnimatedImage(mime, fullURL)

		// Build info line
		infoText := fmt.Sprintf("%dx%d", w, h)
		if isAnim {
			infoText = "ANIM " + infoText
		}
		infoLabel := widget.NewLabel(infoText)
		infoLabel.TextStyle = fyne.TextStyle{Italic: true}
		infoLabel.Alignment = fyne.TextAlignCenter

		// Animated badge overlay
		var content fyne.CanvasObject
		if isAnim {
			// Orange badge for animated
			badgeBg := canvas.NewRectangle(color.RGBA{255, 140, 0, 255})
			badgeBg.SetMinSize(fyne.NewSize(36, 16))
			badgeText := canvas.NewText("ANIM", color.White)
			badgeText.TextSize = 9
			badgeText.TextStyle = fyne.TextStyle{Bold: true}
			badge := container.NewStack(badgeBg, container.NewCenter(badgeText))

			// Position badge at top-left
			badgePos := container.NewHBox(badge)
			content = container.NewBorder(badgePos, infoLabel, nil, nil, imgStack)
		} else {
			content = container.NewBorder(nil, infoLabel, nil, nil, imgStack)
		}

		// Stack with selection border on top
		stack := container.NewStack(content, border)

		// Create tappable wrapper (no hover gray effect)
		tappable := newTappableImage(stack, border, func() {
			// Deselect previous
			if *selectedBorder != nil {
				(*selectedBorder).StrokeWidth = 0
				(*selectedBorder).StrokeColor = color.Transparent
				(*selectedBorder).Refresh()
			}
			// Select this one
			border.StrokeWidth = 3
			border.StrokeColor = color.RGBA{0, 255, 0, 255}
			border.Refresh()
			*selectedBorder = border
			*selectionTarget = fullURL

			typeStr := "Static"
			if isAnim {
				typeStr = "Animated"
			}
			selectedLabel.SetText(fmt.Sprintf("%dx%d %s", w, h, typeStr))
			updatePreview(fullURL, fmt.Sprintf("%dx%d %s - %s", w, h, style, typeStr))
		})

		return tappable
	}

	// Load functions
	loadCapsules := func(appendMode bool) {
		if selectedGameID == 0 {
			return
		}
		if !appendMode {
			capsulePage = 0
			capsuleContainer.RemoveAll()
			selectedCapsuleBorder = nil
		}

		statusLabel.SetText("Loading capsules...")
		go func() {
			grids, err := client.getGrids(selectedGameID, capsuleFilters, capsulePage)
			if err != nil {
				statusLabel.SetText(fmt.Sprintf("Error: %v", err))
				return
			}

			// Filter for portrait orientation (height > width)
			var portraits []sgdbGridData
			for _, g := range grids {
				if g.Height > g.Width {
					portraits = append(portraits, g)
				}
			}

			animCount := 0
			for _, img := range portraits {
				if isAnimatedImage(img.Mime, img.URL) {
					animCount++
				}
				imgData := img
				thumb := createThumb(
					imgData.Thumb, imgData.URL, imgData.Mime,
					fyne.NewSize(120, 180), imgData.Width, imgData.Height, imgData.Style,
					&selectedCapsuleBorder, selectedCapsuleLabel, &selection.GridPortrait,
				)
				capsuleContainer.Add(thumb)
			}
			capsuleContainer.Refresh()

			if len(grids) >= 50 {
				loadMoreCapsule.Show()
			} else {
				loadMoreCapsule.Hide()
			}

			statusLabel.SetText(fmt.Sprintf("Loaded %d capsules (%d animated) - page %d", len(portraits), animCount, capsulePage+1))
			capsulePage++
		}()
	}

	loadWideCapsules := func(appendMode bool) {
		if selectedGameID == 0 {
			return
		}
		if !appendMode {
			widePage = 0
			wideContainer.RemoveAll()
			selectedWideBorder = nil
		}

		statusLabel.SetText("Loading wide capsules...")
		go func() {
			grids, err := client.getGrids(selectedGameID, wideFilters, widePage)
			if err != nil {
				statusLabel.SetText(fmt.Sprintf("Error: %v", err))
				return
			}

			// Filter for landscape orientation (width > height)
			var landscapes []sgdbGridData
			for _, g := range grids {
				if g.Width > g.Height {
					landscapes = append(landscapes, g)
				}
			}

			animCount := 0
			for _, img := range landscapes {
				if isAnimatedImage(img.Mime, img.URL) {
					animCount++
				}
				imgData := img
				thumb := createThumb(
					imgData.Thumb, imgData.URL, imgData.Mime,
					fyne.NewSize(184, 86), imgData.Width, imgData.Height, imgData.Style,
					&selectedWideBorder, selectedWideLabel, &selection.GridLandscape,
				)
				wideContainer.Add(thumb)
			}
			wideContainer.Refresh()

			if len(grids) >= 50 {
				loadMoreWide.Show()
			} else {
				loadMoreWide.Hide()
			}

			statusLabel.SetText(fmt.Sprintf("Loaded %d wide capsules (%d animated) - page %d", len(landscapes), animCount, widePage+1))
			widePage++
		}()
	}

	loadHeroes := func(appendMode bool) {
		if selectedGameID == 0 {
			return
		}
		if !appendMode {
			heroPage = 0
			heroContainer.RemoveAll()
			selectedHeroBorder = nil
		}

		statusLabel.SetText("Loading heroes...")
		go func() {
			heroes, err := client.getHeroes(selectedGameID, heroFilters, heroPage)
			if err != nil {
				statusLabel.SetText(fmt.Sprintf("Error: %v", err))
				return
			}

			animCount := 0
			for _, img := range heroes {
				if isAnimatedImage(img.Mime, img.URL) {
					animCount++
				}
				imgData := img
				thumb := createThumb(
					imgData.Thumb, imgData.URL, imgData.Mime,
					fyne.NewSize(192, 62), imgData.Width, imgData.Height, imgData.Style,
					&selectedHeroBorder, selectedHeroLabel, &selection.HeroImage,
				)
				heroContainer.Add(thumb)
			}
			heroContainer.Refresh()

			if len(heroes) >= 50 {
				loadMoreHero.Show()
			} else {
				loadMoreHero.Hide()
			}

			statusLabel.SetText(fmt.Sprintf("Loaded %d heroes (%d animated) - page %d", len(heroes), animCount, heroPage+1))
			heroPage++
		}()
	}

	loadLogos := func(appendMode bool) {
		if selectedGameID == 0 {
			return
		}
		if !appendMode {
			logoPage = 0
			logoContainer.RemoveAll()
			selectedLogoBorder = nil
		}

		statusLabel.SetText("Loading logos...")
		go func() {
			logos, err := client.getLogos(selectedGameID, logoFilters, logoPage)
			if err != nil {
				statusLabel.SetText(fmt.Sprintf("Error: %v", err))
				return
			}

			for _, img := range logos {
				imgData := img
				thumb := createThumb(
					imgData.Thumb, imgData.URL, imgData.Mime,
					fyne.NewSize(120, 120), imgData.Width, imgData.Height, imgData.Style,
					&selectedLogoBorder, selectedLogoLabel, &selection.LogoImage,
				)
				logoContainer.Add(thumb)
			}
			logoContainer.Refresh()

			if len(logos) >= 50 {
				loadMoreLogo.Show()
			} else {
				loadMoreLogo.Hide()
			}

			statusLabel.SetText(fmt.Sprintf("Loaded %d logos - page %d", len(logos), logoPage+1))
			logoPage++
		}()
	}

	loadIcons := func(appendMode bool) {
		if selectedGameID == 0 {
			return
		}
		if !appendMode {
			iconPage = 0
			iconContainer.RemoveAll()
			selectedIconBorder = nil
		}

		statusLabel.SetText("Loading icons...")
		go func() {
			icons, err := client.getIcons(selectedGameID, iconFilters, iconPage)
			if err != nil {
				statusLabel.SetText(fmt.Sprintf("Error: %v", err))
				return
			}

			for _, img := range icons {
				imgData := img
				thumb := createThumb(
					imgData.Thumb, imgData.URL, imgData.Mime,
					fyne.NewSize(64, 64), imgData.Width, imgData.Height, imgData.Style,
					&selectedIconBorder, selectedIconLabel, &selection.IconImage,
				)
				iconContainer.Add(thumb)
			}
			iconContainer.Refresh()

			if len(icons) >= 50 {
				loadMoreIcon.Show()
			} else {
				loadMoreIcon.Hide()
			}

			statusLabel.SetText(fmt.Sprintf("Loaded %d icons - page %d", len(icons), iconPage+1))
			iconPage++
		}()
	}

	// Load more buttons
	loadMoreCapsule = widget.NewButton("Load More", func() { loadCapsules(true) })
	loadMoreCapsule.Hide()
	loadMoreWide = widget.NewButton("Load More", func() { loadWideCapsules(true) })
	loadMoreWide.Hide()
	loadMoreHero = widget.NewButton("Load More", func() { loadHeroes(true) })
	loadMoreHero.Hide()
	loadMoreLogo = widget.NewButton("Load More", func() { loadLogos(true) })
	loadMoreLogo.Hide()
	loadMoreIcon = widget.NewButton("Load More", func() { loadIcons(true) })
	loadMoreIcon.Hide()

	// Filter panel creator
	createFilterPanel := func(filters *imageFilters, styles, dimensions, mimes []string, onApply func()) fyne.CanvasObject {
		styleSelect := widget.NewSelect(styles, func(s string) {
			if s == "All Styles" {
				filters.style = ""
			} else {
				filters.style = s
			}
		})
		styleSelect.SetSelected("All Styles")

		mimeSelect := widget.NewSelect(mimes, func(s string) {
			if s == "All Formats" {
				filters.mimeType = ""
			} else {
				filters.mimeType = s
			}
		})
		mimeSelect.SetSelected("All Formats")

		animSelect := widget.NewSelect(animationOptions, func(s string) {
			switch s {
			case "Static Only":
				filters.imageType = "static"
			case "Animated Only":
				filters.imageType = "animated"
			default:
				filters.imageType = ""
			}
		})
		animSelect.SetSelected("All")

		dimSelect := widget.NewSelect(dimensions, func(s string) {
			if s == "All Sizes" {
				filters.dimension = ""
			} else {
				filters.dimension = s
			}
		})
		dimSelect.SetSelected("All Sizes")

		nsfwCheck := widget.NewCheck("NSFW", func(b bool) { filters.showNsfw = b })
		humorCheck := widget.NewCheck("Humor", func(b bool) { filters.showHumor = b })
		humorCheck.SetChecked(true)

		applyBtn := widget.NewButtonWithIcon("Apply", theme.SearchIcon(), onApply)

		return container.NewVBox(
			container.NewGridWithColumns(5,
				container.NewVBox(widget.NewLabel("Style:"), styleSelect),
				container.NewVBox(widget.NewLabel("Format:"), mimeSelect),
				container.NewVBox(widget.NewLabel("Animation:"), animSelect),
				container.NewVBox(widget.NewLabel("Size:"), dimSelect),
				container.NewVBox(widget.NewLabel(""), applyBtn),
			),
			container.NewHBox(nsfwCheck, humorCheck),
		)
	}

	// Search list
	searchResultsList := widget.NewList(
		func() int { return len(searchResults) },
		func() fyne.CanvasObject {
			return widget.NewLabel("Game Name Here")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(searchResults) {
				return
			}
			label := obj.(*widget.Label)
			game := searchResults[id]
			text := game.Name
			if game.Verified {
				text += " [Verified]"
			}
			label.SetText(text)
		},
	)

	searchResultsList.OnSelected = func(id widget.ListItemID) {
		if id < len(searchResults) {
			game := searchResults[id]
			selectedGameID = game.ID
			selection.GridDBGameID = game.ID
			gameSelectLabel.SetText(fmt.Sprintf("Selected: %s (ID: %d)", game.Name, game.ID))
			loadCapsules(false)
			loadWideCapsules(false)
			loadHeroes(false)
			loadLogos(false)
			loadIcons(false)
		}
	}

	searchBtn := widget.NewButtonWithIcon("Search", theme.SearchIcon(), func() {
		query := searchEntry.Text
		if query == "" {
			return
		}
		statusLabel.SetText("Searching...")
		go func() {
			results, err := client.search(query)
			if err != nil {
				statusLabel.SetText(fmt.Sprintf("Search error: %v", err))
				return
			}
			searchResults = results
			statusLabel.SetText(fmt.Sprintf("Found %d games", len(searchResults)))
			searchResultsList.Refresh()
		}()
	})

	// Left panel
	searchHeader := container.NewVBox(
		widget.NewLabelWithStyle("Search SteamGridDB", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewBorder(nil, nil, nil, searchBtn, searchEntry),
		gameSelectLabel,
		widget.NewSeparator(),
	)

	leftPanel := container.NewBorder(searchHeader, nil, nil, nil, searchResultsList)

	// Tabs with correct filter options per type
	tabs := container.NewAppTabs(
		container.NewTabItem("Capsule", container.NewBorder(
			container.NewVBox(
				widget.NewLabel("600x900 - Portrait grid"),
				selectedCapsuleLabel,
				createFilterPanel(capsuleFilters, gridStyles, capsuleDimensions, gridMimes, func() { loadCapsules(false) }),
			),
			container.NewCenter(loadMoreCapsule),
			nil, nil,
			container.NewVScroll(capsuleContainer),
		)),
		container.NewTabItem("Wide Capsule", container.NewBorder(
			container.NewVBox(
				widget.NewLabel("920x430 - Horizontal grid"),
				selectedWideLabel,
				createFilterPanel(wideFilters, gridStyles, wideCapsuleDimensions, gridMimes, func() { loadWideCapsules(false) }),
			),
			container.NewCenter(loadMoreWide),
			nil, nil,
			container.NewVScroll(wideContainer),
		)),
		container.NewTabItem("Hero", container.NewBorder(
			container.NewVBox(
				widget.NewLabel("1920x620 - Banner image"),
				selectedHeroLabel,
				createFilterPanel(heroFilters, heroStyles, heroDimensions, gridMimes, func() { loadHeroes(false) }),
			),
			container.NewCenter(loadMoreHero),
			nil, nil,
			container.NewVScroll(heroContainer),
		)),
		container.NewTabItem("Logo", container.NewBorder(
			container.NewVBox(
				widget.NewLabel("Game logo"),
				selectedLogoLabel,
				createFilterPanel(logoFilters, logoStyles, logoDimensions, logoMimes, func() { loadLogos(false) }),
			),
			container.NewCenter(loadMoreLogo),
			nil, nil,
			container.NewVScroll(logoContainer),
		)),
		container.NewTabItem("Icon", container.NewBorder(
			container.NewVBox(
				widget.NewLabel("Square icon"),
				selectedIconLabel,
				createFilterPanel(iconFilters, iconStyles, iconDimensions, iconMimes, func() { loadIcons(false) }),
			),
			container.NewCenter(loadMoreIcon),
			nil, nil,
			container.NewVScroll(iconContainer),
		)),
	)

	// Action buttons
	saveBtn := widget.NewButtonWithIcon("Save Selection", theme.ConfirmIcon(), func() {
		if onSave != nil {
			onSave(selection)
		}
		artWindow.Close()
	})
	saveBtn.Importance = widget.HighImportance

	cancelBtn := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		artWindow.Close()
	})

	clearBtn := widget.NewButtonWithIcon("Clear All", theme.ContentClearIcon(), func() {
		selection.GridPortrait = ""
		selection.GridLandscape = ""
		selection.HeroImage = ""
		selection.LogoImage = ""
		selection.IconImage = ""
		selectedCapsuleLabel.SetText("None selected")
		selectedWideLabel.SetText("None selected")
		selectedHeroLabel.SetText("None selected")
		selectedLogoLabel.SetText("None selected")
		selectedIconLabel.SetText("None selected")
		previewImage.Image = nil
		previewImage.Refresh()
		previewLabel.SetText("Select an image to preview")
		currentPreviewURL = ""
	})

	buttons := container.NewHBox(cancelBtn, clearBtn, saveBtn)

	// Preview panel (right side)
	previewPanel := container.NewVBox(
		widget.NewLabelWithStyle("Preview", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		container.NewCenter(previewImage),
		previewLabel,
		widget.NewSeparator(),
		container.NewCenter(openBrowserBtn),
	)

	// Right preview panel with fixed width
	previewContainer := container.NewVBox(
		previewPanel,
	)

	// Center panel (tabs)
	centerPanel := container.NewBorder(
		nil,
		container.NewVBox(
			widget.NewSeparator(),
			statusLabel,
			container.NewCenter(buttons),
		),
		nil, nil,
		tabs,
	)

	// Main layout: Search | Tabs | Preview
	mainSplit := container.NewHSplit(leftPanel, container.NewHSplit(centerPanel, previewContainer))
	mainSplit.SetOffset(0.18)

	// Set the inner split offset
	if innerSplit, ok := mainSplit.Trailing.(*container.Split); ok {
		innerSplit.SetOffset(0.72)
	}

	artWindow.SetContent(container.NewPadded(mainSplit))
	artWindow.Show()

	if gameName != "" {
		searchBtn.OnTapped()
	}
}

// Cache functions
func GetImageCacheDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = home
	}
	cacheDir := filepath.Join(configDir, "bazzite-devkit", "cache", "images")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}
	return cacheDir, nil
}

func ClearImageCache() error {
	imageCacheMu.Lock()
	imageCache = make(map[string]image.Image)
	imageCacheMu.Unlock()

	cacheDir, err := GetImageCacheDir()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		os.Remove(filepath.Join(cacheDir, entry.Name()))
	}

	return nil
}

func GetCacheSize() (int64, error) {
	cacheDir, err := GetImageCacheDir()
	if err != nil {
		return 0, err
	}

	var size int64
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return 0, err
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		size += info.Size()
	}

	return size, nil
}

func downloadImage(imgURL string) (image.Image, error) {
	// Check memory cache first
	imageCacheMu.RLock()
	if img, ok := imageCache[imgURL]; ok {
		imageCacheMu.RUnlock()
		return img, nil
	}
	imageCacheMu.RUnlock()

	// Check disk cache
	cacheDir, _ := GetImageCacheDir()
	cacheFile := ""
	if cacheDir != "" {
		hash := md5.Sum([]byte(imgURL))
		ext := filepath.Ext(imgURL)
		if ext == "" || len(ext) > 5 {
			ext = ".img"
		}
		cacheFile = filepath.Join(cacheDir, hex.EncodeToString(hash[:])+ext)

		if data, err := os.ReadFile(cacheFile); err == nil {
			if img := decodeImageData(data, imgURL); img != nil {
				imageCacheMu.Lock()
				imageCache[imgURL] = img
				imageCacheMu.Unlock()
				return img, nil
			}
		}
	}

	// Download from URL
	resp, err := http.Get(imgURL)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}

	img := decodeImageData(data, imgURL)
	if img == nil {
		return nil, fmt.Errorf("decode failed for %s", imgURL)
	}

	// Save to disk cache
	if cacheFile != "" {
		os.WriteFile(cacheFile, data, 0644)
	}

	// Save to memory cache
	imageCacheMu.Lock()
	imageCache[imgURL] = img
	imageCacheMu.Unlock()

	return img, nil
}

// decodeImageData tries multiple methods to decode image data
func decodeImageData(data []byte, imgURL string) image.Image {
	if len(data) == 0 {
		return nil
	}

	urlLower := strings.ToLower(imgURL)

	// Check magic bytes for format detection
	isWebP := len(data) > 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP"
	isGIF := len(data) > 6 && (string(data[0:6]) == "GIF87a" || string(data[0:6]) == "GIF89a")
	isPNG := len(data) > 8 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47
	isJPEG := len(data) > 2 && data[0] == 0xFF && data[1] == 0xD8

	// Try WebP (using magic bytes detection)
	if isWebP || strings.Contains(urlLower, ".webp") || strings.Contains(urlLower, "webp") {
		// Try static WebP first (more reliable)
		if img, err := webp.Decode(bytes.NewReader(data)); err == nil {
			return img
		}
		// Try animated WebP
		if webpImg, err := webp.DecodeAll(bytes.NewReader(data)); err == nil && len(webpImg.Image) > 0 {
			return webpImg.Image[0]
		}
	}

	// Try GIF
	if isGIF || strings.HasSuffix(urlLower, ".gif") {
		if gifImg, err := gif.DecodeAll(bytes.NewReader(data)); err == nil && len(gifImg.Image) > 0 {
			firstFrame := gifImg.Image[0]
			bounds := firstFrame.Bounds()
			rgba := image.NewRGBA(bounds)
			draw.Draw(rgba, bounds, firstFrame, bounds.Min, draw.Src)
			return rgba
		}
	}

	// Try PNG/JPEG (standard decode)
	if isPNG || isJPEG || strings.HasSuffix(urlLower, ".png") || strings.HasSuffix(urlLower, ".jpg") || strings.HasSuffix(urlLower, ".jpeg") {
		if img, _, err := image.Decode(bytes.NewReader(data)); err == nil {
			return img
		}
	}

	// Fallback: try all decoders
	// Standard decode first
	if img, _, err := image.Decode(bytes.NewReader(data)); err == nil {
		return img
	}

	// WebP fallback
	if img, err := webp.Decode(bytes.NewReader(data)); err == nil {
		return img
	}
	if webpImg, err := webp.DecodeAll(bytes.NewReader(data)); err == nil && len(webpImg.Image) > 0 {
		return webpImg.Image[0]
	}

	// GIF fallback
	if gifImg, err := gif.DecodeAll(bytes.NewReader(data)); err == nil && len(gifImg.Image) > 0 {
		firstFrame := gifImg.Image[0]
		bounds := firstFrame.Bounds()
		rgba := image.NewRGBA(bounds)
		draw.Draw(rgba, bounds, firstFrame, bounds.Min, draw.Src)
		return rgba
	}

	return nil
}
