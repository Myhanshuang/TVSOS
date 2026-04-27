<script setup>
import MapContainer from '@/components/mapContainer.vue'
import Statistics from '@/components/statistics.vue';
import taskManage from '@/components/taskManage.vue';
import Statistics2 from '@/components/statistics2.vue';
import { watch, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useVisibleStore, useTargetStore } from '@/stores/index.js'

const target = useTargetStore() 
const visible = useVisibleStore()
const scrollContainer = ref(null)

const firstRef = ref(null)
const secondRef = ref(null)
const thirdRef = ref(null)

let observer

onMounted(() => {
  // 1. 修改观察逻辑：使用较高的阈值 (0.6)，确保超过一半进入视口才切换
  // 并增加相互排斥逻辑
  observer = new IntersectionObserver((entries) => {
    // 寻找当前交叉比例最大的那个元素
    const visibleEntry = entries.find(entry => entry.isIntersecting && entry.intersectionRatio > 0.5);
    
    if (visibleEntry) {
      const id = visibleEntry.target.id;
      // 重置所有状态，仅激活当前这一个
      visible.isFirstVisible = (id === 'first');
      visible.isSecondVisible = (id === 'second');
      visible.isThirdVisible = (id === 'third');
    }
  }, { 
    threshold: [0.5, 0.6], // 关键：只有过半时才触发
    rootMargin: "-64px 0px 0px 0px" // 扣除顶部导航栏高度的影响
  }); 

  [firstRef, secondRef, thirdRef].forEach(refItem => {
    if (refItem.value) observer.observe(refItem.value);
  });
});

onUnmounted(() => {
  if (observer) observer.disconnect()
})

// 2. 优化滚动逻辑：使用原生的 scrollIntoView 配合 snap 效果更好
watch(() => target.watchLissoner, async () => {
  if (target.targetId) {
    await nextTick()
    const el = document.getElementById(target.targetId)
    if (el) {
      // 停止 CSS 磁吸干预，执行平滑滚动
      el.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
  }
})
</script>

<template>
  <!-- 整个容器作为滚动根，开启 CSS Snap -->
  <div class="main-snap-container" ref="scrollContainer">
    <div id='first' ref="firstRef" class="section-container">
      <MapContainer/>
    </div>
    <div id='second' ref="secondRef" class="section-container">
      <Statistics2/>
    </div>
    <div id='third' ref="thirdRef" class="section-container">
      <taskManage/>
    </div>
  </div>
</template>

<style scoped>
.main-snap-container {
  height: 100vh;
  overflow-y: scroll;
  scroll-snap-type: y mandatory; /* 核心：垂直方向强制磁吸 */
  scroll-behavior: smooth;
  scrollbar-width: none; /* 隐藏滚动条 (可选) */
}

.main-snap-container::-webkit-scrollbar {
  display: none;
}

.section-container {
  width: 100%;
  height: 100vh; /* 必须是 100vh 以配套磁吸 */
  scroll-snap-align: start; /* 磁吸对齐位置 */
  scroll-snap-stop: always; /* 强制用户一次只滚一屏，防止直接滑过中间组件 */
  box-sizing: border-box;
  overflow: hidden;
}

#first {
  padding-top: 64px; /* 顶部导航栏高度 */
}

#second, #third {
  padding-top: 80px;
}

/* 进场动画优化：基于 visible 状态的简单过渡 */
.section-container > * {
  transition: opacity 0.8s ease, transform 0.8s ease;
  opacity: 0.5;
  transform: scale(0.98);
}

/* 当组件所属的 ID 在 Store 中标记为可见时 */
#first.section-container :deep(#firBorder),
#second.section-container :deep(#secBorder),
#third.section-container :deep(#thiBorder) {
  /* 这里可以根据具体的子组件 ID 或类名微调 */
  opacity: 1;
  transform: scale(1);
}
</style>