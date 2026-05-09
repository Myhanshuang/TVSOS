package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/config"
	"github.com/kiritosuki/mover/internal/constant"
	"github.com/kiritosuki/mover/internal/database"
	"github.com/kiritosuki/mover/internal/model"
)

type RawPoi struct {
	ID   uint    `json:"id"`
	Name string  `json:"name"`
	Lon  float64 `json:"lon"`
	Lat  float64 `json:"lat"`
	Type string  `json:"type"`
}

// MapPoiType 将 JSON 中的中文类型映射为新的 1-11 分类
func MapPoiType(typeName string) int {
	switch typeName {
	case "加油站":
		return constant.PoiTybeGasStation // 1
	case "加气站":
		return constant.PoiTybeCnGStation // 2
	case "其它能源站":
		return constant.PoiTybeOtherEnergy // 3
	case "工厂":
		return constant.PoiTybeFactory // 4
	case "汽车维修", "东风特约维修", "货车维修", "汽车养护/装饰", "长安汽车维修":
		return constant.PoiTybeRepair // 5
	case "物流速递":
		return constant.PoiTybeLogistics // 6
	case "货运火车站":
		return constant.PoiTybeTrainStation // 7
	case "机场货运处":
		return constant.PoiTybeAirport // 8
	case "购物相关场所", "综合市场", "商场", "专卖店", "服装鞋帽皮具店":
		return constant.PoiTybeShopping // 9
	case "家居建材市场":
		return constant.PoiTybeFurniture // 10
	case "公司", "政府机关", "体育休闲服务场所", "生活服务场所":
		return constant.PoiTybeCompany // 11
	default:
		return constant.PoiTybeCompany // 11
	}
}

func main() {
	Logger.InitLogger()

	// 初始化 Viper / Logger 
	// 这里假设我们在 tvsos-backend 路径下执行 go run scripts/import_poi.go
	config.InitViper() 
	
	database.InitDB()

	file, err := os.ReadFile("poi_data/poi.json")
	if err != nil {
		log.Fatalf("读取 poi.json 失败: %v", err)
	}

	var rawPois []RawPoi
	if err := json.Unmarshal(file, &rawPois); err != nil {
		log.Fatalf("解析 JSON 失败: %v", err)
	}

	fmt.Printf("成功读取 %d 条 POI 数据，开始清理并导入数据库...\n", len(rawPois))

	// 修正由于历史字段限制导致的名称过长插入报错 (Data too long for column 'name')
	database.DB.Exec("ALTER TABLE poi MODIFY COLUMN name VARCHAR(255);")

	// 清空表数据
	database.DB.Exec("TRUNCATE TABLE poi")

	var pois []model.Poi
	nameCount := make(map[string]int)

	for _, rp := range rawPois {
		finalName := rp.Name
		if count, exists := nameCount[finalName]; exists {
			// 名称重复，追加序号以保证唯一性
			finalName = fmt.Sprintf("%s_%d", finalName, count+1)
			nameCount[rp.Name] = count + 1
		} else {
			nameCount[finalName] = 1
		}

		pois = append(pois, model.Poi{
			// 如果你想保留原本的 ID：
			// Id:     rp.ID,
			Name:   finalName,
			Lon:    rp.Lon,
			Lat:    rp.Lat,
			Tybe:   MapPoiType(rp.Type),
			Status: constant.PoiStatusWorking,
		})
	}

	result := database.DB.Create(&pois)
	if result.Error != nil {
		log.Fatalf("存入数据库发生异常: %v", result.Error)
	}

	fmt.Printf("成功将 %d 条 POI 导入到数据库中!\n", result.RowsAffected)
}