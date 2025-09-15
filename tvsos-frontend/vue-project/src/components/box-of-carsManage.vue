<!-- 这是小车管理页面的备份 -->

<script setup>
import { onMounted } from 'vue'
import Swiper from 'swiper'
import { Navigation, Grid } from 'swiper/modules'  // ← 从这里引入！
import 'swiper/css'
import 'swiper/css/navigation'
import 'swiper/css/grid'

onMounted(() => {
  new Swiper('.my-swiper', {
    modules: [Navigation, Grid],
    slidesPerView: 3,      // 每行显示 3 个
    grid: {
      rows: 2,             // 两行
      fill: 'row',         // 按行填充
    },
    slidesPerGroup: 3,     // 每次滑动一列
    spaceBetween: 30,
    loop: false,           // 先不循环，保证顺序
    watchSlidesProgress: true,
    watchSlidesVisibility: true, // 开启可见性检测
    speed: 400,
    navigation: {
      nextEl: '.swiper-button-next',
      prevEl: '.swiper-button-prev',
    },
  })
})
</script>

<template>
<div id="thiBorder">
  <div class="hang">
        <div class="space"></div>
      <div class="filterBox"></div>
        <div class="space"></div>
      <div class="searchBox"></div>
        <div class="space"></div>
  </div>

  <div class="swiperBox">
    <div class="swiper my-swiper">
      <div class="swiper-wrapper">
        <div class="swiper-slide" v-for="i in 20" :key="i">
          <div class="box">{{ i }}号元素</div>
        </div>
      </div>
    
      <!-- 左右按钮 -->
      <div class="swiper-button-prev"></div>
      <div class="swiper-button-next"></div>
    </div>
  </div>
</div>
</template>

<style scoped>
#thiBorder{
    margin: 0px;
    padding: 0px;
    width: 99vw;
    height: 88vh;
    border: 1px solid black;
    overflow: hidden;
}

.hang{
  display: flex;
  width: calc(100% - 500px);
  height: 200px;
  margin: 0px;
  padding: 25px 250px;
  border: 1px solid black;
}

.filterBox{
  flex: 300px 3;
  height: 100%;

  border-radius: 25px;
  overflow: hidden;

  box-shadow:  15px 15px 26px #c3c3c3,
         -15px -15px 26px #fdfdfd;
}

.searchBox{
  flex: 200px 2;
  height: 100%;

  border-radius: 25px;
  overflow: hidden;

  box-shadow:  15px 15px 26px #c3c3c3,
               -15px -15px 26px #fdfdfd;
}

.space{
  flex: 1;
  height: 100%;
}

.swiperBox{
  display: inline-block;
  width: calc(100% - 300px);
  height: calc(100% - 300px);
  margin: 0px;
  padding: 25px 150px;
  border: 1px solid black;
}


.my-swiper {
  width: calc(100% - 100px) ;
  height: 100%;
  padding: 0px 50px;
}

.swiper-slide {
  display: flex;
  justify-content: center;
  align-items: center;
}

.box {
  width: calc(100% - 60px);
  height: calc(100% - 10px);
  margin: 0px 30px 0px 30px;
  background: lightblue;
  border-radius: 10px;
  display: flex;
  justify-content: center;
  align-items: center;
}

.swiper-slide {
  opacity: 0;
  transition: all 1.2s cubic-bezier(.35,.74,.33,.75);
}

.swiper-slide-visible {
  opacity: 1;
}

</style>