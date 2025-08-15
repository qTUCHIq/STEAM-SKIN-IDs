package main

import (
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

const (
	byMykelAPIBaseURL = "https://raw.githubusercontent.com/ByMykel/CSGO-API/main/public/api/en/"

	ericZhuAPIBaseURL = "https://raw.githubusercontent.com/EricZhu-42/SteamTradingSite-ID-Mapper/main/"
	counterStrikeJSON = "/730.json"

	modestSerhatAPIBaseURL = "https://raw.githubusercontent.com/ModestSerhat/cs2-marketplace-ids/main/"
)

var (
	defaultHttpClient = &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        20,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     10 * time.Second,
		},
	}

	defaultHeaders = http.Header{
		"User-Agent":      {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36"},
		"Accept":          {"*/*"},
		"Accept-encoding": {"gzip, deflate, br, zstd"},
		"Priority":        {"u=1"},
	}

	defIndexes map[string]int
)

type ModestSerhatResponse struct {
	Items map[string]struct {
		Buff163GoodsID           *int             `json:"buff163_goods_id,omitempty"`
		YoupinID                 *int             `json:"youpin_id,omitempty"`
		BuffMarketGoodsID        *int             `json:"buffmarket_goods_id,omitempty"`
		Buff163StickerID         *int             `json:"buff163_sticker_id,omitempty"`
		Buff163PaintSeedGroupIDs *map[string]*int `json:"buff163_paintseed_group_ids,omitempty"`
		DoublespaceName          *string          `json:"doublespace_name,omitempty"`
		Buff163PhaseIDs          *map[string]*int `json:"buff163_phase_ids,omitempty"`
		Buff163TagIDs            *map[string]*int `json:"buff163_tag_ids,omitempty"`
		Buff163PatchID           *int             `json:"buff163_patch_id,omitempty"`
	} `json:"items"`
	Patterns map[string]map[string][]int `json:"patterns"`
}

type Agent struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DefIndex    string `json:"def_index"`
	Rarity      struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"rarity"`
	Collections []struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Image string `json:"image"`
	} `json:"collections"`
	Team struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"team"`
	MarketHashName string `json:"market_hash_name"`
	Image          string `json:"image"`
	ModelPlayer    string `json:"model_player"`
}

type Collectible struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	DefIndex    string  `json:"def_index"`
	Rarity      struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"rarity"`
	Type           *string `json:"type"`
	Genuine        bool    `json:"genuine"`
	MarketHashName *string `json:"market_hash_name"`
	Image          string  `json:"image"`
}

type Crate struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Description   *string `json:"description"`
	Type          *string `json:"type"`
	FirstSaleDate *string `json:"first_sale_date"`
	Contains      []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Rarity struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Color string `json:"color"`
		} `json:"rarity"`
		PaintIndex string `json:"paint_index"`
		Image      string `json:"image"`
	} `json:"contains"`
	ContainsRare []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Rarity struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Color string `json:"color"`
		} `json:"rarity"`
		PaintIndex *string `json:"paint_index"`
		Phase      *string `json:"phase"`
		Image      string  `json:"image"`
	} `json:"contains_rare"`
	MarketHashName string  `json:"market_hash_name"`
	Rental         bool    `json:"rental"`
	Image          string  `json:"image"`
	ModelPlayer    *string `json:"model_player"`
	LootList       struct {
		Name   string `json:"name"`
		Footer string `json:"footer"`
		Image  string `json:"image"`
	} `json:"loot_list"`
	SpecialNotes []struct {
		Source string `json:"source"`
		Text   string `json:"text"`
	} `json:"special_notes"`
}

type Graffiti struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Rarity      struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"rarity"`
	Crates []struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Image string `json:"image"`
	} `json:"crates"`
	MarketHashName string  `json:"market_hash_name"`
	Image          string  `json:"image"`
	DefIndex       *string `json:"def_index"`
	SpecialNotes   []struct {
		Source string `json:"source"`
		Text   string `json:"text"`
	} `json:"special_notes"`
}

type Highlight struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	TournamentEvent  string `json:"tournament_event"`
	Team0            string `json:"team0"`
	Team1            string `json:"team1"`
	Stage            string `json:"stage"`
	TournamentPlayer string `json:"tournament_player"`
	Map              string `json:"map"`
	MarketHashName   string `json:"market_hash_name"`
	Image            string `json:"image"`
	Video            string `json:"video"`
}

type Keychain struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	DefIndex    *string `json:"def_index"`
	Rarity      struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"rarity"`
	Collections []struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Image string `json:"image"`
	} `json:"collections"`
	MarketHashName string `json:"market_hash_name"`
	Image          string `json:"image"`
	Highlight      bool   `json:"highlight"`
}

type Key struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Crates      []struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Image string `json:"image"`
	} `json:"crates"`
	MarketHashName *string `json:"market_hash_name"`
	Marketable     bool    `json:"marketable"`
	Image          string  `json:"image"`
}

type MusicKit struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DefIndex    string `json:"def_index"`
	Rarity      struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"rarity"`
	MarketHashName *string `json:"market_hash_name"`
	Exclusive      bool    `json:"exclusive"`
	Image          string  `json:"image"`
}

type Patch struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DefIndex    string `json:"def_index"`
	Rarity      struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"rarity"`
	MarketHashName string `json:"market_hash_name"`
	Image          string `json:"image"`
}

type Skin struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Weapon      struct {
		ID       string `json:"id"`
		WeaponID int    `json:"weapon_id"`
		Name     string `json:"name"`
	} `json:"weapon"`
	Category struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"category"`
	Pattern *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"pattern"`
	MinFloat *float64 `json:"min_float"`
	MaxFloat *float64 `json:"max_float"`
	Rarity   struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"rarity"`
	Stattrak   bool    `json:"stattrak"`
	Souvenir   bool    `json:"souvenir"`
	PaintIndex *string `json:"paint_index"`
	Wears      []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"wears"`
	Collections []struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Image string `json:"image"`
	} `json:"collections"`
	Crates []struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Image string `json:"image"`
	} `json:"crates"`
	Team struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"team"`
	LegacyModel  bool   `json:"legacy_model"`
	Image        string `json:"image"`
	Phase        string `json:"phase"`
	SpecialNotes []struct {
		Source string `json:"source"`
		Text   string `json:"text"`
	} `json:"special_notes"`
}

type Sticker struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DefIndex    string `json:"def_index"`
	Rarity      struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"rarity"`
	Crates []struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Image string `json:"image"`
	} `json:"crates"`
	TournamentEvent  string  `json:"tournament_event"`
	Type             string  `json:"type"`
	MarketHashName   *string `json:"market_hash_name"`
	Effect           string  `json:"effect"`
	Image            string  `json:"image"`
	TournamentTeam   string  `json:"tournament_team"`
	TournamentPlayer string  `json:"tournament_player"`
	SpecialNotes     []struct {
		Source string `json:"source"`
		Text   string `json:"text"`
	} `json:"special_notes"`
}

func getRequest(url string, target any) error {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("Failed to create request for URL %s: %w", url, err)
	}

	request.Header = defaultHeaders

	response, err := defaultHttpClient.Do(request)
	if err != nil {
		return fmt.Errorf("Request execution failed for URL %s: %w", url, err)
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Unexpected status code for URL %s: %d", url, response.StatusCode)
	}

	bodyReader, err := getDecompressedBody(response)
	if err != nil {
		return fmt.Errorf("Failed to get decompressed body from response for URL %s: %w", url, err)
	}

	defer bodyReader.Close()

	body, err := io.ReadAll(bodyReader)
	if err != nil {
		return fmt.Errorf("Failed to read response body for URL %s: %w", url, err)
	}

	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("Failed to unmarshal response body for URL %s: %w", url, err)
	}

	return nil
}

func getDecompressedBody(response *http.Response) (io.ReadCloser, error) {
	switch response.Header.Get("Content-Encoding") {

	case "gzip":
		reader, err := gzip.NewReader(response.Body)
		if err != nil {
			return nil, fmt.Errorf("Failed to create gzip reader: %w", err)
		}
		return reader, nil

	case "deflate":
		reader, err := zlib.NewReader(response.Body)
		if err != nil {
			return nil, fmt.Errorf("Failed to create deflate reader: %w", err)
		}
		return reader, nil

	case "br":
		return io.NopCloser(brotli.NewReader(response.Body)), nil

	case "zstd":
		decoder, err := zstd.NewReader(response.Body)
		if err != nil {
			return nil, fmt.Errorf("Failed to create zstd decoder: %w", err)
		}
		return io.NopCloser(decoder), nil

	default:
		return response.Body, nil
	}
}

func getSteamIndexes(endpoint string) (map[string]int, map[string]int, error) {
	url := byMykelAPIBaseURL + endpoint

	var data []Skin
	if err := getRequest(url, &data); err != nil {
		return nil, nil, fmt.Errorf("Failed to fetch steam indexes. %w", err)
	}

	defIndexes := make(map[string]int, len(data))
	paintIndexes := make(map[string]int, len(data))

	for _, item := range data {
		defIndexes[item.Weapon.Name] = item.Weapon.WeaponID
		if item.Pattern != nil {
			paintIndex, err := strconv.Atoi(*item.PaintIndex)
			if err == nil {
				paintIndexes[item.Pattern.Name] = paintIndex
			}
		}
	}

	return defIndexes, paintIndexes, nil
}

func getSteamAgentIDs(endpoint string) (map[string]int, error) {
	url := byMykelAPIBaseURL + endpoint

	var data []Agent
	if err := getRequest(url, &data); err != nil {
		return nil, fmt.Errorf("Failed to fetch steam ids. %w", err)
	}

	ids := make(map[string]int, len(data))

	for _, item := range data {
		id, err := strconv.Atoi(item.DefIndex)

		if err == nil {
			ids[item.MarketHashName] = id
		}
	}

	return ids, nil
}

func getSteamCollectibleIDs(endpoint string) (map[string]int, error) {
	url := byMykelAPIBaseURL + endpoint

	var data []Collectible
	if err := getRequest(url, &data); err != nil {
		return nil, fmt.Errorf("Failed to fetch steam ids. %w", err)
	}

	ids := make(map[string]int, len(data))
	excludedPattern := `\b(Souvenir Token|2024 Souvenir Package|2025 Souvenir Package)\b`

	for _, item := range data {
		id, err := strconv.Atoi(item.DefIndex)

		if err == nil {
			marketHashName := item.MarketHashName
			if marketHashName != nil {
				isExcluded, _ := regexp.MatchString(excludedPattern, *marketHashName)
				if !isExcluded {
					ids[*marketHashName] = id
				}
			}
		}
	}

	return ids, nil
}

func getSteamCrateIDs(endpoint string) (map[string]int, error) {
	url := byMykelAPIBaseURL + endpoint

	var data []Crate
	if err := getRequest(url, &data); err != nil {
		return nil, fmt.Errorf("Failed to fetch steam ids. %w", err)
	}

	ids := make(map[string]int, len(data))
	excludedPattern := `\b(Sticker Collection|Patch Collection|Storage Unit)\b`

	for _, item := range data {
		idParts := strings.Split(item.ID, "-")

		id, err := strconv.Atoi(idParts[1])
		if err == nil {
			marketHashName := item.MarketHashName
			isExcluded, _ := regexp.MatchString(excludedPattern, marketHashName)
			if !isExcluded {
				ids[marketHashName] = id
			}
		}
	}

	return ids, nil
}

func getSteamGraffitiIDs(endpoint string) (map[string]string, error) {
	url := byMykelAPIBaseURL + endpoint

	var data []Graffiti
	if err := getRequest(url, &data); err != nil {
		return nil, fmt.Errorf("Failed to fetch steam ids. %w", err)
	}

	ids := make(map[string]string, len(data))
	excludedPattern := `\b(2019|2021|2022|2023|2024|2025)\b`

	for _, item := range data {
		idParts := strings.Split(item.ID, "-")
		marketHashName := item.MarketHashName

		isExcluded, _ := regexp.MatchString(excludedPattern, marketHashName)
		if !isExcluded {
			ids[marketHashName] = idParts[1]
		}
	}

	return ids, nil
}

func getSteamHighlightIDs(endpoint string) (map[string]string, error) {
	url := byMykelAPIBaseURL + endpoint

	var data []Highlight
	if err := getRequest(url, &data); err != nil {
		return nil, fmt.Errorf("Failed to fetch steam ids. %w", err)
	}

	ids := make(map[string]string, len(data))

	for _, item := range data {
		ids[item.MarketHashName] = item.ID
	}

	return ids, nil
}

func getSteamKeychainIDs(endpoint string) (map[string]int, error) {
	url := byMykelAPIBaseURL + endpoint

	var data []Keychain
	if err := getRequest(url, &data); err != nil {
		return nil, fmt.Errorf("Failed to fetch steam ids. %w", err)
	}

	ids := make(map[string]int, len(data))

	for _, item := range data {
		if item.DefIndex != nil {
			id, err := strconv.Atoi(*item.DefIndex)

			if err == nil {
				ids[item.MarketHashName] = id
			}
		}
	}

	return ids, nil
}

func getSteamKeyIDs(endpoint string) (map[string]any, error) {
	url := byMykelAPIBaseURL + endpoint

	var data []Key
	if err := getRequest(url, &data); err != nil {
		return nil, fmt.Errorf("Failed to fetch steam ids. %w", err)
	}

	ids := make(map[string]any, len(data))

	for _, item := range data {
		marketHashName := item.MarketHashName

		if marketHashName != nil {
			idParts := strings.Split(item.ID, "-")

			id, err := strconv.Atoi(idParts[1])
			if err == nil {
				ids[*marketHashName] = id
			} else {
				ids[*marketHashName] = idParts[1]
			}
		}
	}

	return ids, nil
}

func getSteamMusicKitIDs(endpoint string) (map[string]int, error) {
	url := byMykelAPIBaseURL + endpoint

	var data []MusicKit
	if err := getRequest(url, &data); err != nil {
		return nil, fmt.Errorf("Failed to fetch steam ids. %w", err)
	}

	ids := make(map[string]int, len(data))

	for _, item := range data {
		id, err := strconv.Atoi(item.DefIndex)

		if err == nil {
			marketHashName := item.MarketHashName
			if marketHashName != nil {
				ids[*marketHashName] = id
			}
		}
	}

	return ids, nil
}

func getSteamPatchIDs(endpoint string) (map[string]int, error) {
	url := byMykelAPIBaseURL + endpoint

	var data []Patch
	if err := getRequest(url, &data); err != nil {
		return nil, fmt.Errorf("Failed to fetch steam ids. %w", err)
	}

	ids := make(map[string]int, len(data))

	for _, item := range data {
		id, err := strconv.Atoi(item.DefIndex)

		if err == nil {
			ids[item.MarketHashName] = id
		}
	}

	return ids, nil
}

func getSteamStickerIDs(endpoint string) (map[string]int, error) {
	url := byMykelAPIBaseURL + endpoint

	var data []Sticker
	if err := getRequest(url, &data); err != nil {
		return nil, fmt.Errorf("Failed to fetch steam ids. %w", err)
	}

	excludedNames := map[string]struct{}{
		"Sticker | 3DMAX | DreamHack 2014":             {},
		"Sticker | London Conspiracy | DreamHack 2014": {},
		"Sticker | dAT team | DreamHack 2014":          {},
		"Sticker | mousesports | DreamHack 2014":       {},
		"Sticker | Ninja (Foil)":                       {},
		"Sticker | The Bomber (Foil)":                  {},
		"Sticker | The Nader (Foil)":                   {},
		"Sticker | The Awper (Foil)":                   {},
		"Sticker | The Fragger (Foil)":                 {},
		"Sticker | Support (Foil)":                     {},
		"Sticker | The Leader (Foil)":                  {},
	}

	ids := make(map[string]int, len(data))

	for _, item := range data {
		id, err := strconv.Atoi(item.DefIndex)

		if err == nil {
			marketHashName := item.MarketHashName
			if marketHashName != nil {
				if _, isExcluded := excludedNames[*marketHashName]; !isExcluded {
					ids[*marketHashName] = id
				}
			}
		}
	}

	return ids, nil
}

func getSteamMarketIDs(marketplace string) (map[string]int, error) {
	url := ericZhuAPIBaseURL + marketplace + counterStrikeJSON

	var data map[string]struct {
		CnName string `json:"cn_name"`
		EnName string `json:"en_name"`
		NameID int    `json:"name_id"`
	}

	if err := getRequest(url, &data); err != nil {
		return nil, fmt.Errorf("Failed to fetch market ids. %w", err)
	}

	ids := make(map[string]int, len(data))

	for name, item := range data {
		enName := item.EnName
		if enName == name || strings.HasSuffix(enName, "(Holo/Foil)") {
			if _, exists := defIndexes[name]; !exists {
				ids[name] = item.NameID
			}
		}
	}

	return ids, nil
}

func getChineseMarketIDs(marketplace string) (map[string]int, error) {
	url := ericZhuAPIBaseURL + marketplace + counterStrikeJSON

	var data map[string]int
	if err := getRequest(url, &data); err != nil {
		return nil, fmt.Errorf("Failed to fetch chinese market ids. %w", err)
	}

	for name, id := range data {
		if id == -1 {
			delete(data, name)
		}
		if _, exists := defIndexes[name]; exists {
			delete(data, name)
		}
	}

	return data, nil
}

func getModestSerhatIDs(marketplace string) (map[string]int, map[string]int, map[string]map[string]int, map[string]map[string]int, map[string]map[string]int, map[string]int, map[string]map[string][]int, error) {
	url := modestSerhatAPIBaseURL + marketplace

	var data ModestSerhatResponse
	if err := getRequest(url, &data); err != nil {
		return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("Failed to fetch buff market ids. %w", err)
	}

	buffMarketIDs := make(map[string]int, len(data.Items))
	buff163StickerIDs := make(map[string]int, len(data.Items))
	buff163PaintseedGroupIDs := make(map[string]map[string]int)
	buff163PhaseIDs := make(map[string]map[string]int)
	buff163TagIDs := make(map[string]map[string]int)
	buff163PatchIDs := make(map[string]int, len(data.Items))

	for name, item := range data.Items {
		if id := item.BuffMarketGoodsID; id != nil {
			buffMarketIDs[name] = *id
		}

		if id := item.Buff163StickerID; id != nil {
			buff163StickerIDs[name] = *id
		}

		if groups := item.Buff163PaintSeedGroupIDs; groups != nil {
			m := buff163PaintseedGroupIDs[name]
			if m == nil {
				m = make(map[string]int, len(*groups))
				buff163PaintseedGroupIDs[name] = m
			}
			for group, id := range *groups {
				if id != nil {
					m[group] = *id
				}
			}
		}

		if phases := item.Buff163PhaseIDs; phases != nil {
			m := buff163PhaseIDs[name]
			if m == nil {
				m = make(map[string]int, len(*phases))
				buff163PhaseIDs[name] = m
			}
			for phase, id := range *phases {
				if id != nil {
					m[phase] = *id
				}
			}
		}

		if tags := item.Buff163TagIDs; tags != nil {
			m := buff163TagIDs[name]
			if m == nil {
				m = make(map[string]int, len(*tags))
				buff163TagIDs[name] = m
			}
			for tag, id := range *tags {
				if id != nil {
					m[tag] = *id
				}
			}
		}

		if id := item.Buff163PatchID; id != nil {
			buff163PatchIDs[name] = *id
		}
	}

	return buffMarketIDs, buff163StickerIDs, buff163PaintseedGroupIDs, buff163PhaseIDs, buff163TagIDs, buff163PatchIDs, data.Patterns, nil
}

func saveData[T any](data map[string]T, filePath string, isPretty bool) {
	if data == nil {
		return
	}

	sortedKeys := make([]string, 0, len(data))
	for key := range data {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	sortedDataMap := make(map[string]any, len(data))
	for _, key := range sortedKeys {
		sortedDataMap[key] = data[key]
	}

	file, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("Failed to create file %s: %s\n", filePath, err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	if isPretty {
		encoder.SetIndent("", "    ")
	}

	err = encoder.Encode(sortedDataMap)
	if err != nil {
		fmt.Printf("Failed to encode data to JSON: %s\n", err)
	}
}

func saveDataAsync[T any](wg *sync.WaitGroup, data map[string]T, basePath string) {
	miniPath := "./mini/" + basePath
	prettyPath := "./pretty/" + basePath

	wg.Add(2)
	go func() {
		defer wg.Done()
		saveData(data, miniPath, false)
	}()
	go func() {
		defer wg.Done()
		saveData(data, prettyPath, true)
	}()
}

func main() {
	dirs := []string{
		"./mini/buff163_grouped_ids",
		"./mini/steam_grouped_ids",
		"./mini/steam_indexes",
		"./mini/market_ids",
		"./pretty/buff163_grouped_ids",
		"./pretty/steam_grouped_ids",
		"./pretty/steam_indexes",
		"./pretty/market_ids",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			fmt.Printf("Failed to create directory %s: %v\n", dir, err)
			return
		}
	}

	var paintIndexes map[string]int
	var steamAgentIDs map[string]int
	var steamCollectibleIDs map[string]int
	var steamCrateIDs map[string]int
	var steamGraffitiIDs map[string]string
	var steamHighlightIDs map[string]string
	var steamKeychainIDs map[string]int
	var steamKeyIDs map[string]any
	var steamMusicKitIDs map[string]int
	var steamPatchIDs map[string]int
	var steamStickerIDs map[string]int
	var buffMarketIDs map[string]int
	var buff163StickerIDs map[string]int
	var buff163PaintseedGroupIDs map[string]map[string]int
	var buff163PhaseIDs map[string]map[string]int
	var buff163TagIDs map[string]map[string]int
	var buff163PatchIDs map[string]int
	var buff163Patterns map[string]map[string][]int

	errs := make(chan error, 12)
	var wg sync.WaitGroup
	wg.Add(12)

	go func() {
		defer wg.Done()
		var err error
		defIndexes, paintIndexes, err = getSteamIndexes("skins.json")
		if err != nil {
			errs <- err
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		steamAgentIDs, err = getSteamAgentIDs("agents.json")
		if err != nil {
			errs <- err
			return
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		steamCollectibleIDs, err = getSteamCollectibleIDs("collectibles.json")
		if err != nil {
			errs <- err
			return
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		steamCrateIDs, err = getSteamCrateIDs("crates.json")
		if err != nil {
			errs <- err
			return
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		steamGraffitiIDs, err = getSteamGraffitiIDs("graffiti.json")
		if err != nil {
			errs <- err
			return
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		steamHighlightIDs, err = getSteamHighlightIDs("highlights.json")
		if err != nil {
			errs <- err
			return
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		steamKeychainIDs, err = getSteamKeychainIDs("keychains.json")
		if err != nil {
			errs <- err
			return
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		steamKeyIDs, err = getSteamKeyIDs("keys.json")
		if err != nil {
			errs <- err
			return
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		steamMusicKitIDs, err = getSteamMusicKitIDs("music_kits.json")
		if err != nil {
			errs <- err
			return
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		steamPatchIDs, err = getSteamPatchIDs("patches.json")
		if err != nil {
			errs <- err
			return
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		steamStickerIDs, err = getSteamStickerIDs("stickers.json")
		if err != nil {
			errs <- err
			return
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		buffMarketIDs, buff163StickerIDs, buff163PaintseedGroupIDs, buff163PhaseIDs, buff163TagIDs, buff163PatchIDs, buff163Patterns, err = getModestSerhatIDs("cs2_marketplaceids.json")
		if err != nil {
			errs <- err
			return
		}
	}()

	wg.Wait()
	close(errs)

	for err := range errs {
		fmt.Println("Error during API fetch. ", err)
	}

	var steamMarketIDs map[string]int
	chineseMarketplaces := []string{"buff", "c5", "uuyp", "igxe"}
	chineseMarketIDs := make(map[string]map[string]int)

	var mu sync.Mutex
	errs = make(chan error, 5)
	wg.Add(5)

	go func() {
		defer wg.Done()
		var err error
		steamMarketIDs, err = getSteamMarketIDs("steam")
		if err != nil {
			errs <- err
			return
		}
	}()

	for _, marketplace := range chineseMarketplaces {
		go func(marketplace string) {
			defer wg.Done()
			ids, err := getChineseMarketIDs(marketplace)
			if err != nil {
				errs <- err
				return
			}
			mu.Lock()
			chineseMarketIDs[marketplace] = ids
			mu.Unlock()
		}(marketplace)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		fmt.Println("Error during API fetch. ", err)
	}

	saveDataAsync(&wg, defIndexes, "steam_indexes/def_indexes.json")
	saveDataAsync(&wg, paintIndexes, "steam_indexes/paint_indexes.json")

	saveDataAsync(&wg, steamAgentIDs, "steam_grouped_ids/agents.json")
	saveDataAsync(&wg, steamCollectibleIDs, "steam_grouped_ids/collectibles.json")
	saveDataAsync(&wg, steamCrateIDs, "steam_grouped_ids/crates.json")
	saveDataAsync(&wg, steamGraffitiIDs, "steam_grouped_ids/graffiti.json")
	saveDataAsync(&wg, steamHighlightIDs, "steam_grouped_ids/highlights.json")
	saveDataAsync(&wg, steamKeychainIDs, "steam_grouped_ids/keychains.json")
	saveDataAsync(&wg, steamKeyIDs, "steam_grouped_ids/keys.json")
	saveDataAsync(&wg, steamMusicKitIDs, "steam_grouped_ids/music_kits.json")
	saveDataAsync(&wg, steamPatchIDs, "steam_grouped_ids/patches.json")
	saveDataAsync(&wg, steamStickerIDs, "steam_grouped_ids/stickers.json")

	saveDataAsync(&wg, steamMarketIDs, "market_ids/steam.json")
	saveDataAsync(&wg, chineseMarketIDs["buff"], "market_ids/buff163.json")
	saveDataAsync(&wg, chineseMarketIDs["c5"], "market_ids/c5game.json")
	saveDataAsync(&wg, chineseMarketIDs["uuyp"], "market_ids/youpin898.json")
	saveDataAsync(&wg, chineseMarketIDs["igxe"], "market_ids/igxe.json")
	saveDataAsync(&wg, buffMarketIDs, "market_ids/buff_market.json")

	saveDataAsync(&wg, buff163StickerIDs, "buff163_grouped_ids/stickers.json")
	saveDataAsync(&wg, buff163PaintseedGroupIDs, "buff163_grouped_ids/paintseed_group_ids.json")
	saveDataAsync(&wg, buff163PhaseIDs, "buff163_grouped_ids/phases.json")
	saveDataAsync(&wg, buff163TagIDs, "buff163_grouped_ids/tags.json")
	saveDataAsync(&wg, buff163PatchIDs, "buff163_grouped_ids/patches.json")
	saveDataAsync(&wg, buff163Patterns, "buff163_grouped_ids/patterns.json")

	wg.Wait()
}
