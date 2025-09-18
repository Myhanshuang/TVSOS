<template>
    <div class="guide-page" @wheel="handleWheel">
        <div class="guide-content" :class="{ 'slide-up': isSliding }">
            <div class="logo">
                <img src="@/assets/images/logo.png" alt="Logo">
            </div>
            <h1>欢迎使用 TVSOS</h1>
            <p>运输车辆调度优化与仿真</p>

            <div class="slide-hint" @mouseenter="pauseAnimation" @mouseleave="resumeAnimation" @click="handleClick">
                <div class="arrow-animation"><img src="../assets/images/向下箭头.png" alt="向下滑动以登录" class="arrow"></div>
                <span>向下滑动以登录</span>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const isSliding = ref(false)

// 在setup函数中添加两个新方法
const animationPaused = ref(false)

const pauseAnimation = () => {
    const arrow = document.querySelector('.arrow-animation')
    if (arrow) {
        arrow.style.animationPlayState = 'paused'
        animationPaused.value = true
    }
}

const resumeAnimation = () => {
    const arrow = document.querySelector('.arrow-animation')
    if (arrow) {
        arrow.style.animationPlayState = 'running'
        animationPaused.value = false
    }
}
const handleWheel = (event) => {
    if (isSliding.value) return;

    // 向下滚动时触发
    if (event.deltaY > 0) {
        isSliding.value = true
        navigateToLogin()
    }
}

const navigateToLogin = () => {
    // 添加滑动动画
    const content = document.querySelector('.guide-content')
    if (content) {
        content.style.opacity = '0'
        content.style.transform = 'translateY(-50px)'
        content.style.transition = 'all 0.5s ease'
    }

    setTimeout(() => {
        router.push('/login')
    }, 500)
}

// 添加点击事件支持（可选）
const handleClick = () => {
    if (!isSliding.value) {
        isSliding.value = true
        navigateToLogin()
    }
}
</script>

<style scoped>
.guide-page {
    height: 100vh;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    display: flex;
    align-items: center;
    justify-content: center;
    overflow: hidden;
    position: relative;
    cursor: pointer;
    /* 显示可点击光标 */
}

.guide-content {
    text-align: center;
    color: white;
    transition: all 0.8s ease;
    /* 统一过渡效果 */
    padding: 20px;
}

.guide-content.slide-up {
    transform: translateY(-100vh);
    opacity: 0;
}

.logo {
    margin-bottom: 30px;
}

.logo img {
    width: 100px;
    height: 100px;
    border-radius: 20px;
}

h1 {
    font-size: 2.8rem;
    margin-bottom: 15px;
    font-weight: 700;
    background: linear-gradient(135deg,
        #ffffff 0%,
        #f8f8f8 15%,
        #e6e6e6 25%,
        #d0d0d0 35%,
        #c0c0c0 45%,
        #b0b0b0 50%,
        #c0c0c0 55%,
        #d0d0d0 65%,
        #e6e6e6 75%,
        #f8f8f8 85%,
        #ffffff 100%);
    -webkit-background-clip: text;
    background-clip: text;
    color: transparent;
    text-shadow:
        0 2px 4px rgba(255, 255, 255, 0.8),
        0 -1px 1px rgba(0, 0, 0, 0.3),
        0 4px 6px rgba(255, 255, 255, 0.5),
        0 -2px 3px rgba(0, 0, 0, 0.2);
    position: relative;
    display: inline-block;
    padding: 5px 0;
}

h1::before {
    content: '';
    position: absolute;
    top: -3px;
    left: -10px;
    right: -10px;
    bottom: -3px;
    background: linear-gradient(45deg, 
        rgba(255,255,255,0) 0%, 
        rgba(255,255,255,0.4) 50%, 
        rgba(255,255,255,0) 100%);
    z-index: -1;
    transform: skewX(-15deg);
    animation: shine 3s infinite;
    opacity: 0.7;
}
@keyframes shine {
    0% { left: -100%; }
    100% { left: 100%; }
}
p {
    font-size: 1.4rem;
    opacity: 0.95;
    margin-bottom: 50px;
    font-weight: 300;
    letter-spacing: 0.5px;
}

.slide-hint {
    margin-top: 40px;
    padding: 20px;
    border-radius: 15px;
    background: rgba(255, 255, 255, 0.15);
    cursor: pointer;
    transition: all 0.3s ease;
}

.slide-hint:hover {
    background: rgba(255, 255, 255, 0.25);
    transform: translateY(-5px);
}

.slide-hint span {
    display: block;
    margin-top: 10px;
    font-size: 1.1rem;
    opacity: 0.8;
    font-weight: 400;
}

.arrow {
    width: 30px;
    height: 30px;

}

.arrow-animation {
    animation: bounce 2s infinite;
    display: flex;
    justify-content: center;
    align-items: center;
    animation-play-state: running;
}

@keyframes bounce {

    0%,
    20%,
    50%,
    80%,
    100% {
        transform: translateY(0);
    }

    40% {
        transform: translateY(-10px);
    }

    60% {
        transform: translateY(-5px);
    }
}
</style>