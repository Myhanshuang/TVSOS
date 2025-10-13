<script setup>
import MapContainer from '@/components/mapContainer.vue'
import Statistics from '@/components/statistics.vue';
import carsManage from '@/components/carsManage.vue';

import { watch, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useVisibleStore, useTargetStore, useImformStore } from '@/stores/index.js'

const imform = useImformStore()
const target = useTargetStore() 
const visible =useVisibleStore()

// // 👇 三个状态
// const isFirstVisible = ref(false)
// const isSecondVisible = ref(false)
// const isThirdVisible = ref(false)

const firstRef = ref(null)
const secondRef = ref(null)
const thirdRef = ref(null)
let observer

onMounted(() => {
  observer = new IntersectionObserver(
    (entries) => {
      entries.forEach(entry => {
        if (entry.target.id === 'first') {
          visible.isFirstVisible = entry.isIntersecting
        } else if (entry.target.id === 'second') {
          visible.isSecondVisible = entry.isIntersecting
        } else if (entry.target.id === 'third') {
          visible.isThirdVisible = entry.isIntersecting
        }
      })
    },
    {
      // window 滚动用 null
      root: null,       
      // 10% 可见就触发
      threshold: 0.1    
    }
  )

  if (firstRef.value) observer.observe(firstRef.value)
  if (secondRef.value) observer.observe(secondRef.value)
  if (thirdRef.value) observer.observe(thirdRef.value)
})

onUnmounted(() => {
  if (observer) {
    observer.disconnect()
  }
})


// 简化版 cubic-bezier 贝塞尔函数生成器
function cubicBezier(p1x, p1y, p2x, p2y) {
  return function (t) {
    const u = 1 - t
    // 三次贝塞尔公式
    return (3 * u * u * t * p1y) +
           (3 * u * t * t * p2y) +
           (t * t * t)
  }
}


// watch监测Lissoner是否有点击函数，触发翻页函数

watch(() => target.watchLissoner, async () => {
  if (target.targetId) {
    await nextTick()
    const el = document.getElementById(target.targetId)
    if (el) {
      const targetTop = el.getBoundingClientRect().top + window.scrollY
      const startTop = window.scrollY
      const distance = targetTop - startTop
      // 动画总时长控制
      const duration = 350
      const startTime = performance.now()

      // 应用贝塞尔缓降函数
      const easing = cubicBezier(1,.01,.99,.01)

      function step(currentTime) {
        const elapsed = currentTime - startTime
        const progress = Math.min(elapsed / duration, 1)
        const eased = easing(progress)
        window.scrollTo(0, startTop + distance * eased)

        if (progress < 1) {
          requestAnimationFrame(step)
        }
      }

      requestAnimationFrame(step)
    }
  }
})
</script>

<template>
  <div id='first' ref="firstRef" :class="{Component: 1, isVisible: visible.isFirstVisible}">
    <MapContainer/>
  </div>
  <div id='second' ref="secondRef" :class="{Component: 1, isVisible: visible.isSecondVisible}">
    <Statistics/>
  </div>
  <div id='third' ref="thirdRef" :class="{Component: 1, isVisible: visible.isThirdVisible}">
    <carsManage/>
  </div>
</template>

<style scoped>
.Component{
  margin: 0px 0px 140px 0px;
  padding: 60px 0px 0px 0px;

  transform: translateY(10px);
  opacity: 0;
  transition: all 0.6s;
}

.isVisible{
  transform: translateY(0px);
  opacity: 1;
}
</style>
