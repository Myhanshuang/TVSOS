<script setup>
import { onMounted, onUnmounted } from "vue";
import AMapLoader from "@amap/amap-jsapi-loader";

import { useImformStore } from '@/stores'
import { getPOIList } from '@/api/poi'

import gasStationIcon from '@/assets/images/加油站.png'
import gasIcon from '@/assets/images/加气站.png'
import energyIcon from '@/assets/images/其他能源站.png'
import factoryIcon from '@/assets/images/工厂.png'
import companyEnterpriseIcon from '@/assets/images/公司企业.png'
import shoppingCenterIcon from '@/assets/images/购物中心.png'
import furnitureBuildingMaterialsMarketIcon from '@/assets/images/家具建材市场.png'
// 地图样例
let map = null;
let labelsLayer = null;
// pinia store访问函数
let imform = useImformStore()


onMounted(() => {
  window._AMapSecurityConfig = {
    securityJsCode: "09582d73da9c81d93b134caf4e6f173a",
  };

  AMapLoader.load({
    key: "84a1985a18fcdb13254b2d85d69885ee",
    version: "2.0",
    plugins: ["AMap.Scale", "AMap.ToolBar", "AMap.MoveAnimation", "AMap.LabelsLayer"],
  })
    .then((AMap) => {
      map = new AMap.Map("mapContainer", {
        viewMode: "3D",
        zoom: 13,
        center: [104.065861, 30.6574013],
        mapStyle: "amap://styles/whitesmoke"
      });

      map.addControl(new AMap.ToolBar());
      map.addControl(new AMap.Scale());
      labelsLayer = new AMap.LabelsLayer({
        zooms: [3, 20],
        zIndex: 1000,
        collision: true, // 该层内标注是否避让
        allowCollision: true, // 不同标注层之间是否避让  
      });
      map.add(labelsLayer);
      getPOIList()
        .then(response => {
          console.log('接口返回:', response)
          if (response.data && response.data.code === 1 && response.data.data && response.data.data.length > 0) {
            createLabelMarkers(response.data.data, AMap);
          } else {
            console.warn("数据为空:", response.data);
          }
        })
        .catch(error => {
          console.error("获取POI数据失败:", error);
        });
    })
    .catch((e) => {
      console.error("地图加载失败:", e);
    });

});
// 创建 LabelMarker 标记
function createLabelMarkers(data, AMap) {
  const labelMarkers = [];

  data.forEach((item, index) => {
    // 根据类型设置不同的图标和颜色
    let iconUrl;
    switch (item.type) {
      case 1: // 加油站
        iconUrl = gasStationIcon;
        break;
      case 2: // 加气站
        iconUrl = gasIcon;
        break;
      case 3: // 其它能源站
        iconUrl = energyIcon;
        break;
      case 4: // 工厂
        iconUrl = factoryIcon;
        break;
      case 5: // 公司企业
        iconUrl = companyEnterpriseIcon;
        break;
      case 6: // 购物中心
        iconUrl = shoppingCenterIcon;
        break;
      case 7: // 家具建材市场
        iconUrl = furnitureBuildingMaterialsMarketIcon;
        break;
      default:
        iconUrl = gasStationIcon;
    }

    // 设置图标对象
    const icon = {
      type: "image",
      image: iconUrl,
      size: [24, 24], // 图标尺寸
      anchor: "center",
    };



    // 创建 LabelMarker
    const labelMarker = new AMap.LabelMarker({
      name: `poi_${item.id}`,
      position: [item.lon, item.lat],
      zIndex: 10 + index,
      rank: index,
      icon: icon,

    });

    // 添加点击事件
    labelMarker.on('click', function (e) {
      console.log('点击了标记:', item);
      // 可以在这里触发显示详细信息的面板
      // imform.showInfo(item);

      // 地图视口移动居中
      map.setZoomAndCenter(15,[item.lon, item.lat]);
    });

    // 添加鼠标悬停事件
    labelMarker.on('mouseover', function (e) {
      labelMarker.setOpacity(0.8);
    });

    labelMarker.on('mouseout', function (e) {
      labelMarker.setOpacity(1);
    });

    labelMarkers.push(labelMarker);
  });

  console.log('创建的标记数量:', labelMarkers.length);

  // 批量添加 labelMarker 到图层
  labelsLayer.add(labelMarkers);

  // 调整地图视野以包含所有标记
  if (labelMarkers.length > 0) {
    map.setFitView();
  }
}
// 组件卸载时清理
onUnmounted(() => {
  if (labelsLayer) {
    labelsLayer.clear();
    labelsLayer = null;
  }
  if (map) {
    map.destroy(); // 彻底销毁地图实例
    map = null;
    console.log("地图已销毁");
  }
});

</script>



<template>
  <div id="firBorder">
    <div id="mapBox" :class="{ wideMap: !imform.imformIf, shrotMap: imform.imformIf }">
      <div id="mapContainer"></div>
    </div>


    <div id="carImfromBox" :class="{ imformShow: imform.imformIf, imformHide: !imform.imformIf }">
      <div id="imfromBox" :class="{ show: imform.imformIf, hide: !imform.imformIf }">
        这里是小车的信息
      </div>
    </div>
  </div>
</template>

<style scoped>
#firBorder {
  margin: 0;
  padding: 0px 0px;
  width: 100vw;
  height: 92vh;
  display: inline-block;
  text-align: left;

  z-index: 1;
}

#mapBox {
  margin: 0px;
  padding: 0px;
  width: 100%;
  height: 100%;
  display: inline-block;
  position: relative;
  vertical-align: top;
  z-index: 2;
}

#carImfromBox {
  display: inline-block;
  position: absolute;
  right: 2vw;
  top: 13vh;
  vertical-align: top;
  width: 20vw;
  height: 80vh;

  border-radius: 20px;
  background: white;
  box-shadow: 14px 14px 30px #bebebe,
    -14px -14px 30px #ffffff;

  z-index: 3;
}

#imformBox {
  display: inline-block;
  background-color: aliceblue;
  height: calc(100% - 20px);
  width: calc(100% - 10px);
  margin: 10px 10px 10px 10px;
}

.imformHide {
  margin: 0px;
  padding: 0px;
  width: 0px;
  opacity: 0;

  transform: translateX(200px);

  transition: all 0.4s cubic-bezier(.35, .74, .33, .75) 0.4s;
}

.imformShow {
  margin: 0px 0px 0px 100px;
  border-radius: 25px;
  width: 300px;
  opacity: 1;

  transform: translateX(0px);

  transition: all 0.4s cubic-bezier(.35, .74, .33, .75);

}

.show {
  transform: translateX(0px);
  opacity: 1;
  transition: all 0.4s cubic-bezier(.35, .74, .33, .75) 0.4s;
}

.hide {
  transform: translateX(50px);
  opacity: 0;
  transition: all 0.4s cubic-bezier(.35, .74, .33, .75);
}

#mapContainer {
  width: 100%;
  height: 100%;
}
</style>