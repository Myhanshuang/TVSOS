<script setup>
import MapContainer from '@/components/mapContainer.vue'
import Statistics from '@/components/statistics.vue';
import taskManage from '@/components/taskManage.vue';
import { watch, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useVisibleStore, useTargetStore } from '@/stores/index.js'

const target = useTargetStore() 
const visible =useVisibleStore()
const firstRef = ref(null)
const secondRef = ref(null)
const thirdRef = ref(null)

let observer

onMounted(() => {
  observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
      const id = entry.target.id;
      const key = `is${id.charAt(0).toUpperCase() + id.slice(1)}Visible`;
      visible[key] = entry.isIntersecting;
    });
  }, { threshold: 0.1 });

  [firstRef, secondRef, thirdRef].forEach(refItem => {
    if (refItem.value) observer.observe(refItem.value);
  });
});

onUnmounted(() => {
  if (observer) {
    observer.disconnect()
  }
})


function cubicBezier(p1x, p1y, p2x, p2y) {
  return function (t) {
    const u = 1 - t
    return (3 * u * u * t * p1y) +
           (3 * u * t * t * p2y) +
           (t * t * t)
  }
}

watch(() => target.watchLissoner, async () => {
  if (target.targetId) {
    await nextTick()
    const el = document.getElementById(target.targetId)
    if (el) {
      const targetTop = el.getBoundingClientRect().top + window.scrollY
      const startTop = window.scrollY
      const distance = targetTop - startTop
      const duration = 350
      const startTime = performance.now()
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
    <taskManage/>
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
