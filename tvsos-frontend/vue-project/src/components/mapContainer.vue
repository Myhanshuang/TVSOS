<script setup>
import { onMounted, onUnmounted } from "vue";
import AMapLoader from "@amap/amap-jsapi-loader";
import { useImformStore } from '@/stores/imform'


// 地图样例
let map = null;

// pinia store访问函数
let imform = useImformStore()


onMounted(() => {
  window._AMapSecurityConfig = {
    securityJsCode: "09582d73da9c81d93b134caf4e6f173a",
  };

  AMapLoader.load({
    key: "84a1985a18fcdb13254b2d85d69885ee",
    version: "2.0",
    plugins: ["AMap.Scale", "AMap.ToolBar", "AMap.MoveAnimation"],
  })
    .then((AMap) => {
      map = new AMap.Map("mapContainer", {
        viewMode: "3D",
        zoom: 13,
        center: [104.065861, 30.6574013],
      });

      map.addControl(new AMap.ToolBar());
      map.addControl(new AMap.Scale());
      map.setFitView();
    })
    .catch((e) => {
      console.error("地图加载失败:", e);
    });
});

// 组件卸载时清理
onUnmounted(() => {
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


  <div id="carImfromBox" :class="{ imformShow: imform.imformIf, imformHide: !imform.imformIf}">
    <div id="imfromBox" :class="{ show: imform.imformIf, hide: !imform.imformIf}">
      这里是小车的信息
    </div>
  </div>
</div>
</template>

<style scoped>
#firBorder{
  margin: 0;
  padding: 0px 0px;
  width: 100vw;
  height: 92vh;
  display: inline-block;
  text-align: left;

  z-index: 1;
}

#mapBox{
  margin: 0px;
  padding: 0px;
  width: 100%;
  height: 100%;
  display: inline-block;
  position: relative;
  vertical-align: top;
  z-index: 2;
}

#carImfromBox{
  display: inline-block;
  position: absolute;
  right: 2vw;
  top: 13vh;
  vertical-align: top;
  width: 20vw;
  height: 80vh;

  border-radius: 20px;
  background: #f7f7f7;
  box-shadow:  14px 14px 30px #bebebe,
             -14px -14px 30px #ffffff;

  z-index: 3;
}

#imformBox{
  display: inline-block;
  background-color: aliceblue;
  height: calc(100% - 20px);
  width: calc(100% - 10px);
  margin: 10px 10px 10px 10px;
}

.imformHide{
  margin: 0px;
  padding: 0px;
  width: 0px;
  opacity: 0;

  transform: translateX(200px);

  transition: all 0.4s cubic-bezier(.35,.74,.33,.75) 0.4s;
}

.imformShow{
  margin: 0px 0px 0px 100px;
  border-radius: 25px;
  width: 300px;
  opacity: 1;

  transform: translateX(0px);

  transition: all 0.4s cubic-bezier(.35,.74,.33,.75);

}

.show{
  transform: translateX(0px);
  opacity: 1;
  transition: all 0.4s cubic-bezier(.35,.74,.33,.75) 0.4s;
}

.hide{
  transform: translateX(50px);
  opacity: 0;
  transition: all 0.4s cubic-bezier(.35,.74,.33,.75);
}

#mapContainer{
  width: 100%;
  height: 100%;

  /* margin: 3% 3%; */
}
</style>