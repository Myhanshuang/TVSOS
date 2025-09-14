<script setup>
import { useImformStore } from '@/stores/imform'; 
import { useTargetStore } from '@/stores/scrollTarget'

const imform = useImformStore()
const target = useTargetStore()

</script>


<template>
<nav>
    <el-button @click="imform.imformChange">小车详细信息触发按钮</el-button>
    <el-button @click="target.targetChange('first')">主页</el-button>
    <el-button @click="target.targetChange('second')">统计</el-button>
    <el-button @click="target.targetChange('third')">小车管理</el-button>
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
    /* 高德地图的logo的z-index很高 */
    z-index: 1000;

    background: #ffffff;

    /* 导航栏阴影设置 */
    box-shadow:  5px 5px 6px #a8a8a8,
                -5px -5px 6px #ffffff;
}
</style>