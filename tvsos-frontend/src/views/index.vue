<script setup>
import MapContainer from '@/components/mapContainer.vue'
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

const sectionIds = ['first', 'second', 'third']

let observer
let onKeydown

/** 横向滚动到指定页（下标 0/1/2） */
const scrollToSection = (index) => {
  const el = scrollContainer.value
  if (!el) return
  el.scrollTo({ left: index * window.innerWidth, behavior: 'smooth' })
}

/** 相对当前页左右切换（dir=±1） */
const switchByDelta = (dir) => {
  const el = scrollContainer.value
  if (!el) return
  const currentIndex = Math.round(el.scrollLeft / window.innerWidth)
  const next = currentIndex + dir
  if (next >= 0 && next < sectionIds.length) scrollToSection(next)
}

onMounted(() => {
  // 1. 横向可见性观察：超过一半进入视口才激活对应页面
  observer = new IntersectionObserver((entries) => {
    const visibleEntry = entries.find(entry => entry.isIntersecting && entry.intersectionRatio > 0.5);
    if (visibleEntry) {
      const id = visibleEntry.target.id;
      visible.isFirstVisible = (id === 'first');
      visible.isSecondVisible = (id === 'second');
      visible.isThirdVisible = (id === 'third');
    }
  }, { threshold: [0.5, 0.6] });

  [firstRef, secondRef, thirdRef].forEach(refItem => {
    if (refItem.value) observer.observe(refItem.value);
  });

  // 2. 左右方向键切页
  onKeydown = (e) => {
    if (e.key === 'ArrowRight') switchByDelta(1)
    else if (e.key === 'ArrowLeft') switchByDelta(-1)
  }
  window.addEventListener('keydown', onKeydown)
});

onUnmounted(() => {
  if (observer) observer.disconnect()
  if (onKeydown) window.removeEventListener('keydown', onKeydown)
})

// 导航按钮触发：横向平滑对齐目标页
watch(() => target.watchLissoner, async () => {
  if (target.targetId) {
    await nextTick()
    const idx = sectionIds.indexOf(target.targetId)
    if (idx >= 0) scrollToSection(idx)
  }
})
</script>

<template>
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
  <!-- 左右翻页箭头 -->
  <div class="page-arrow arrow-left" @click="switchByDelta(-1)">‹</div>
  <div class="page-arrow arrow-right" @click="switchByDelta(1)">›</div>
</template>

<style scoped>
.main-snap-container {
  height: 100vh;
  display: flex;
  overflow-x: auto;
  overflow-y: hidden;
  scroll-snap-type: x mandatory; /* 横向强制磁吸 */
  scroll-behavior: smooth;
  scrollbar-width: none; /* 隐藏滚动条 (可选) */
}

.main-snap-container::-webkit-scrollbar {
  display: none;
}

.section-container {
  width: 100vw;
  min-width: 100vw;
  height: 100vh;
  flex-shrink: 0;
  scroll-snap-align: start; /* 磁吸对齐位置 */
  scroll-snap-stop: always; /* 强制一次只滚一屏 */
  box-sizing: border-box;
  overflow-y: auto;  /* 每页纵向自由滚动，不再截断 */
  overflow-x: hidden;
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
  opacity: 1;
  transform: scale(1);
}

/* 左右翻页箭头 */
.page-arrow {
  position: fixed;
  top: 50%;
  transform: translateY(-50%);
  width: 44px;
  height: 44px;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.35);
  color: #fff;
  font-size: 28px;
  line-height: 44px;
  text-align: center;
  cursor: pointer;
  z-index: 999;
  user-select: none;
  transition: background 0.3s;
}
.page-arrow:hover {
  background: rgba(24, 144, 255, 0.8);
}
.arrow-left { left: 8px; }
.arrow-right { right: 8px; }
</style>