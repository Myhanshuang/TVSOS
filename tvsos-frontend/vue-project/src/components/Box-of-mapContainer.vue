<!-- 这是用flex弹性盒子做出来的废稿，v-if无法实现动画的过度 -->

<script setup>
import { onMounted } from "vue";
import AMapLoader from "@amap/amap-jsapi-loader";
import { useCounterStore } from '@/stores/counter'


// 地图样例
let map = null;

let counter = useCounterStore()


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

</script>



<template>
<div id="border">
  <div id="mapBox">
    <div id="mapContainer"></div>
  </div>

<transition name="imform">
  <div id="carImfromBox" v-if="counter.imformIf === 'imformShow'">

  </div>
</transition>
</div>
</template>

<style scoped>
#border{
  border: 1px solid black;

  margin: 0;
  padding: 10px 5vw;
  width: 90vw;
  height: 85vh;
  display: flex;
  align-items: center;
  justify-content: center;

  z-index: 1;
}

#mapBox{
  border: 1px solid black;
  border-radius: 25px;

  margin: 0px 0px 0px 3vw;
  padding: 10px 10px;
  display: flex;
  
  height: calc(100% - 20px);
  /* flex: 4 500px; */

  transition: all 1s;
}

#mapBox {
  flex-basis: 1700px;
  transition: flex-basis 1s ease;
}

#mapBox.expand {
  flex-basis: 2400px;
}

#carImfromBox{
  border: 1px solid black;
  border-radius: 25px;

  margin: 0px 0px 0px 5vw;
  /* flex: 200px; */
    flex-basis: 600px;
  height: 100%;
}




.imform-enter-from,
.imform-leave-to {
  opacity: 0;
  transform: translateX(100px);
}

.imform-enter-active,
.imform-leave-active {
  transition: all 1s ease;
}

.imform-enter-to,
.imform-leave-from {
  opacity: 1;
  transform: translateX(0px);
}

.imformHide{
  opacity: 0;

  transform: translateX(100px);
  transition: all 1s;
}

.imformShow{
  transform: translateX(0px);
}




#mapContainer{
  width: 100%;
  height: 100%;
}
</style>