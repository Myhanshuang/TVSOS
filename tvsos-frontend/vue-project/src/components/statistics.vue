<script setup>
import { onMounted } from 'vue'
import Swiper from 'swiper'
import 'swiper/css'
import 'swiper/css/navigation' // 导航按钮的样式

import { Navigation } from 'swiper/modules'

import { useVisibleStore } from '@/stores'

const visible = useVisibleStore()

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
    <div id=""></div>
    <!-- <svg xmlns="@/assets/images/wave" viewBox="0 0 1440 320">
      <path fill="#a2d9ff" fill-opacity="1"
        d="M0,32L48,80C96,128,192,224,288,224C384,224,480,128,576,122.7C672,117,768,203,864,202.7C960,203,1056,117,1152,90.7C1248,64,1344,96,1392,112L1440,128L1440,320L1392,320C1344,320,1248,320,1152,320C1056,320,960,320,864,320C768,320,672,320,576,320C480,320,384,320,288,320C192,320,96,320,48,320L0,320Z">
      </path>
    </svg> -->
    <div id="cardsBox">
      <div :class="{ card: 1, cardShow: !visible.isSecondVisible, hoverBox: 1 }">
        这是一个卡片
      </div>
      <div class="space"></div>
      <div :class="{ card: 1, cardShow: !visible.isSecondVisible, hoverBox: 1 }">
        这是一个卡片
      </div>
      <div class="space"></div>
      <div :class="{ card: 1, cardShow: !visible.isSecondVisible, hoverBox: 1 }">
        这是一个卡片
      </div>
    </div>
    <div class="swiper">
      <div class="swiper-wrapper">
        <div class="swiper-slide chartCard hoverBox">
          <div class="box">1号</div>
        </div>
        <div class="swiper-slide chartCard hoverBox">
          <div class="box">2号</div>
        </div>
        <div class="swiper-slide chartCard hoverBox">
          <div class="box">3号</div>
        </div>
        <div class="swiper-slide chartCard hoverBox">
          <div class="box">4号</div>
        </div>
        <div class="swiper-slide chartCard hoverBox">
          <div class="box">5号</div>
        </div>
        <div class="swiper-slide chartCard hoverBox">
          <div class="box">6号</div>
        </div>
      </div>

      <!-- 左右按钮 -->
      <div class="swiper-button-prev"></div>
      <div class="swiper-button-next"></div>
    </div>
  </div>
</template>


<style scoped>
#secBorder {
  margin: 0px;
  padding: 0px;
  width: 99vw;
  height: 88vh;

  /* border: 1px solid black; */
  overflow: hidden;

}

#cardsBox {
  display: flex;
  justify-content: center;
  /* 水平居中 */
  align-items: center;
  /* 垂直居中 */
  margin: 0px;
  padding: 50px 250px;
  width: calc(100% - 500px);
  height: 200px;

  /* border: 1px solid black; */
}

.card {
  display: inline-block;
  height: calc(100% - 20px);
  width: 220px;
  padding: 10px 15px;
  border-radius: 20px;
  color: #355174;
  font-family: 'Segoe UI', 'Arial', 'PingFang SC', 'Hiragino Sans GB', '微软雅黑', sans-serif;
  border-radius: 45px;
  background: aliceblue;
  box-shadow: 15px 15px 26px #c3c3c3,
    -15px -15px 26px #fdfdfd;
  transition: all 0.5s;
}

.space {
  flex-grow: 1;
}

.swiper {
  width: calc(100% - 500px);
  height: calc(100% - 400px);
  margin: 0px;
  padding: 50px 250px;

}

.swiper-slide {
  opacity: 0;
  transition: all 1s cubic-bezier(.35, .74, .33, .75);
}

.swiper-slide-visible {
  opacity: 1;
}

.chartCard {
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
  background: aliceblue;

}




.box {
  display: inline-block;
  width: calc(100% - 40px);
  height: calc(100% - 20px);
  padding: 10px 20px;
  margin: 0px;
  color: #355174;
  font-family: 'Segoe UI', 'Arial', 'PingFang SC', 'Hiragino Sans GB', '微软雅黑', sans-serif;
}

.hoverBox:hover {
  box-shadow: 0 15px 20px rgba(0, 0, 0, 0.2),
    0 30px 40px rgba(0, 0, 0, 0.1);
  transform: translateY(-3px);
}

/* 淡入动画 */
.cardShow {
  opacity: 0;
  transform: translateY(50px);
}
</style>
