<script setup>
import { onMounted } from 'vue'
import Swiper from 'swiper'
import 'swiper/css'
import 'swiper/css/navigation' // 导航按钮的样式

import { Navigation } from 'swiper/modules'


onMounted(() => {
  const swiper = new Swiper('.swiper', {
    modules: [Navigation],
    slidesPerView: 2,   // 每次显示 2 个元素
    spaceBetween: 60,   // 每个卡片间距
    loop: true,           // 开启循环
    watchSlidesProgress: true,
    watchSlidesVisibility: true, // 开启可见性检测
    speed: 400,
    navigation: {
      nextEl: '.swiper-button-next',
      prevEl: '.swiper-button-prev',
    }
  })
})
</script>

<template>
<div id="secBorder">
    <div id="cardsBox">
        <div class="card">
            这是一个卡片
        </div>
        <div class="space"></div>
        <div class="card">
            这是一个卡片
        </div>
        <div class="space"></div>
        <div class="card">
            这是一个卡片
        </div>
    </div>
    <div class="swiper">
      <div class="swiper-wrapper">
        <div class="swiper-slide chartCard"><div class="box">1号</div></div>
        <div class="swiper-slide chartCard"><div class="box">2号</div></div>
        <div class="swiper-slide chartCard"><div class="box">3号</div></div>
        <div class="swiper-slide chartCard"><div class="box">4号</div></div>
        <div class="swiper-slide chartCard"><div class="box">5号</div></div>
        <div class="swiper-slide chartCard"><div class="box">6号</div></div>
      </div>
    
      <!-- 左右按钮 -->
      <div class="swiper-button-prev"></div>
      <div class="swiper-button-next"></div>
    </div>
</div>
</template>


<style scoped>
#secBorder{
    margin: 0px;
    padding: 0px;
    width: 99vw;
    height: 88vh;

    /* border: 1px solid black; */
    overflow: hidden;

}

#cardsBox{
    display: flex;
    justify-content: center; /* 水平居中 */
    align-items: center;     /* 垂直居中 */
    margin: 0px;
    padding: 50px 250px;
    width: calc(100% - 500px);
    height: 200px;

    /* border: 1px solid black; */
}

.card{
    display: inline-block;
    height: calc(100% - 20px);
    width: 220px;
    padding: 10px 15px;
    border-radius: 20px;

    border-radius: 45px;
    background: linear-gradient(145deg, #f6f6f6, #cacaca);
    box-shadow:  15px 15px 26px #c3c3c3,
             -15px -15px 26px #fdfdfd;
}

.space{
    flex-grow: 1;
}

.swiper{
    width: calc(100% - 500px);
    height: calc(88vh - 400px);
    margin: 0px;
    padding: 50px 250px;

}

.swiper-slide {
  opacity: 0;
  transition: all 1s cubic-bezier(.35,.74,.33,.75);
}

.swiper-slide-visible {
  opacity: 1;
}

.chartCard{
    display: flex;
    justify-content: center;
    align-items: center;
    height: 100%;
    /* 必要的元素间距 */
    /* 否则会导致第三个元素的显现与初始化时的意外移动 */
    margin: 0px 60px 0px 0px;
    width: auto;
    border-radius: 50px;
    overflow: hidden;
    background: linear-gradient(145deg, #f6f6f6, #cacaca);
    box-shadow:  15px 15px 26px #c3c3c3,
             -15px -15px 26px #fdfdfd;

}

.box{
    display: inline-block;
    width: calc(100% - 40px);
    height: calc(100% - 20px);
    padding: 10px 20px;
    margin: 0px;
}

</style>
