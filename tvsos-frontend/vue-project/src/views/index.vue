<script setup>
import MapContainer from '@/components/mapContainer.vue'
import Statistics from '@/components/statistics.vue';
import carsManage from '@/components/carsManage.vue';

import { useImformStore } from '@/stores/imform.js'
import { useTargetStore } from '@/stores/scrollTarget'
import { watch, nextTick } from 'vue'

const imform = useImformStore()
const target = useTargetStore() 

watch(() => target.watchLissoner, async () => {
  if (target.targetId) {
    await nextTick()
    const el = document.getElementById(target.targetId)
    if (el) {
      el.scrollIntoView({
        behavior: 'smooth',
        block: 'start'
      })
    }
  }
})


</script>

<template>
    <div id='first' class="Component">
        <MapContainer/>
    </div>
    <div id='second' class="Component">
        <Statistics/>
    </div>
    <div id='third' class="Component">
        <carsManage/>
    </div>
</template>

<style scoped>
.Component{
    margin: 0px;
    padding: 80px 0px 40px 0px;
}
</style>