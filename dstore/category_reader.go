package dstore

import (
	"bufio"
	"encoding/json"
	"os"
)

type AppJSONInfo struct {
	Id         string
	Category   string
	Name       string
	LocaleName map[string]string
}

func getCategoryInfo(file string) (map[string]string, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoder := json.NewDecoder(bufio.NewReader(f))
	jsonData := map[string]AppJSONInfo{}
	if err := decoder.Decode(&jsonData); err != nil {
		return nil, err
	}

	infos := map[string]string{}
	for k, v := range jsonData {
		infos[k] = v.Category
	}
	return infos, nil
}

func getXCategoryInfo(file string) (map[string]string, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	infos := map[string]string{}
	decoder := json.NewDecoder(bufio.NewReader(f))
	if err := decoder.Decode(&infos); err != nil {
		return nil, err
	}

	return infos, nil
}

type CategoryJSONInfo struct {
	ID      string
	Locales map[string]map[string]string
	Name    string
}

func GetAllInfos(file string) []CategoryJSONInfo {
	fallbackCategories := []CategoryJSONInfo{
		CategoryJSONInfo{
			ID:   OthersName,
			Name: OthersName,
		},
		CategoryJSONInfo{
			ID:   InternetName,
			Name: InternetName,
		},
		CategoryJSONInfo{
			ID:   OfficeName,
			Name: OfficeName,
		},
		CategoryJSONInfo{
			ID:   DevelopmentName,
			Name: DevelopmentName,
		},
		CategoryJSONInfo{
			ID:   ReadingName,
			Name: ReadingName,
		},
		CategoryJSONInfo{
			ID:   GraphicsName,
			Name: GraphicsName,
		},
		CategoryJSONInfo{
			ID:   GameName,
			Name: GameName,
		},
		CategoryJSONInfo{
			ID:   MusicName,
			Name: MusicName,
		},
		CategoryJSONInfo{
			ID:   SystemName,
			Name: SystemName,
		},
		CategoryJSONInfo{
			ID:   VideoName,
			Name: VideoName,
		},
		CategoryJSONInfo{
			ID:   ChatName,
			Name: ChatName,
		},
	}
	f, err := os.Open(file)
	if err != nil {
		return fallbackCategories
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	var jsonInfo []CategoryJSONInfo
	if err := decoder.Decode(&jsonInfo); err != nil {
		return fallbackCategories
	}

	categoryInfos := make([]CategoryJSONInfo, len(jsonInfo))
	for i, info := range jsonInfo {
		categoryInfos[i] = info
	}

	return categoryInfos
}
