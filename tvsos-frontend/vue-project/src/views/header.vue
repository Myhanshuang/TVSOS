<script setup>

import { useVisibleStore, useTargetStore, useImformStore } from '@/stores'
import { ref } from 'vue'
import PoiList from '@/components/poiList.vue' 

const imform = useImformStore()
const target = useTargetStore()
const visible = useVisibleStore()
const drawerVisible = ref(false)
const openDrawer = () => {
    drawerVisible.value = true
}
</script>


<template>
    <nav>
        <div class="title">TVSOS</div>
        <div class="space"></div>
        <el-button @click="target.targetChange('first')"
            :class="{ scollButton: 1, scollButtonActive: visible.isFirstVisible }">地图</el-button>
        <div class="space"></div>
        <el-button @click="target.targetChange('second')"
            :class="{ scollButton: 1, scollButtonActive: visible.isSecondVisible }">统计</el-button>
        <div class="space"></div>
        <el-button @click="target.targetChange('third')"
            :class="{ scollButton: 1, scollButtonActive: visible.isThirdVisible }">小车管理</el-button>
        <el-button @click="imform.imformChange">小车详细信息触发按钮</el-button>
        <el-button type="primary" @click="openDrawer"> poi列表</el-button>
        <div class="longSpace"></div>
        <div class="loginOutBox">
            <router-link to="/login" class="loginOut">退出登录</router-link>
        </div>
    </nav>
    <poi-list :visible="drawerVisible" @update:visible="drawerVisible = $event" />
    <!-- 监听 poi-list 组件触发的 update:visible 事件，当事件触发时，将事件传递的值（$event）赋值给父组件的 drawerVisible 变量 -->
    <RouterView></RouterView>
</template>

<style scoped>
nav {
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
    display: flex;
    align-items: center;
    justify-items: center;

    /* 导航栏阴影设置 */
    box-shadow: 5px 5px 6px #a8a8a8,
        -5px -5px 6px #ffffff;
}

.title {
    display: flex;
    align-items: center;
    justify-items: center;
    height: 100%;
    width: auto;
    padding: 0px 20px;

    font-size: 20px;
    font-family: 'Courier New', Courier, monospace;
}

.scollButton {
    height: 100%;
    width: auto;
    border: 0;
    padding: 0px 20px;
    margin: 0px 0px 0px 20px;
    position: relative;
    font-size: 18px;
}

.scollButton:hover {
    background-color: #ffffff;
    color: rgb(126, 173, 249);
}

.scollButton::after {
    content: " ";
    position: absolute;
    display: inline-block;
    width: 0px;
    height: 3px;
    background-color: rgb(126, 173, 249);
    bottom: 13px;
    transition: all 0.3s;
}

.scollButton:hover::after {
    width: 55%;
}

.scollButtonActive {
    background-color: #ffffff;
    color: rgb(126, 173, 249);
}


.scollButtonActive::after {
    width: 50%;
}

.space {
    display: inline-block;
    height: 100%;
    flex: 1;
}

.longSpace {
    display: inline-block;
    height: 100%;
    flex: 100;
}

.loginOutBox {
    display: flex;
    align-items: center;
    justify-items: center;
    height: 100%;
    width: auto;
    padding: 0px 20px;
}

.loginOut {
    color: rgb(97, 98, 102);
    font-size: 18px;
    font-family: 'Franklin Gothic Medium', 'Arial Narrow', Arial, sans-serif;

    text-decoration: none;
    transition: all 0.5s;
}

.loginOut:hover {
    color: rgb(255, 0, 0);
}
</style>