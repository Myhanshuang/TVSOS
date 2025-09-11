<script setup>
import { useCounterStore } from '@/stores/counter'; 

const counter = useCounterStore()
</script>


<template>
<nav>
    <el-button :onclick="counter.imformChange">小车详细信息触发按钮</el-button>
</nav>
    <RouterView></RouterView>
</template>

<style scoped>
nav{
    /* 使导航栏固定于页面上边框 */
    position: fixed;
    top: 0;
    margin: 0px;
    padding: 0px;

    height: 60px;
    width: 100%;

    /* 使导航栏渲染在其他主件上方 */
    z-index: 10;

    background: #ffffff;

    /* 导航栏阴影设置 */
    box-shadow:  5px 5px 6px #a8a8a8,
                -5px -5px 6px #ffffff;
}
</style>