<template>
    <el-row class="login-page">
        <el-col :span="12" class="bg"></el-col>
        <el-col :span="6" :offset="3" class="form">
            <el-form v-if="isRegister" ref="form" size="large" autocomplete="off" :model="formModel" :rules="rules">
                <el-form-item>
                    <h1>注册</h1>
                </el-form-item>

                <el-form-item prop="email">
                    <el-input :prefix-icon="Message" placeholder="请输入邮箱" v-model="formModel.email"></el-input>
                </el-form-item>
                <el-form-item prop="password">
                    <el-input name="password" :prefix-icon="Lock" type="password" placeholder="请输入密码"
                        v-model="formModel.password" show-password></el-input>
                </el-form-item>
                <el-form-item prop="repassword">
                    <el-input name="password" :prefix-icon="Lock" type="password" placeholder="请再次输入密码"
                        v-model="formModel.repassword" show-password></el-input>
                </el-form-item>
                <el-form-item v-if="isRegister">
                    <el-button class="button" type="primary" auto-insert-space @click="register">注册</el-button>
                </el-form-item>
                <el-form-item v-if="isRegister">
                    <el-link type="primary" :underline="false" @click="isRegister = false">← 返回</el-link>
                </el-form-item>
            </el-form>
            <el-form v-else ref="form" size="large" autocomplete="off" :model="formModel" :rules="rules">
                <el-form-item>
                    <h1>登录</h1>
                </el-form-item>
                <el-form-item prop="email">
                    <el-input :prefix-icon="Message" placeholder="请输入邮箱" v-model="formModel.email"></el-input>
                </el-form-item>
                <el-form-item prop="password">
                    <el-input name="password" :prefix-icon="Lock" type="password" placeholder="请输入密码"
                        v-model="formModel.password"></el-input>
                </el-form-item>
                <el-form-item class="flex">
                    <div class="flex">
                        <el-checkbox>记住我</el-checkbox>
                        <el-link type="primary" :underline="false"
                            @click="router.push('/forgotpassword')">忘记密码？</el-link>
                    </div>
                </el-form-item>
                <el-form-item>
                    <el-button class="button" type="primary" auto-insert-space @click="login">登录</el-button>
                </el-form-item>
                <el-form-item>
                    <el-link type="primary" :underline="false" @click="isRegister = true">注册 →</el-link>
                </el-form-item>
            </el-form>
        </el-col>
    </el-row>
</template>

<script setup>
import { ElMessage } from 'element-plus';
import { ref, watch } from 'vue';
import { useUserStore } from '@/stores';
import router from '@/router';

const isRegister = ref(false)
const form = ref(null)
const formModel = ref({
    email: '',
    password: '',
    repassword: ''
})

const register = async () => {
    await form.value.validate()
    ElMessage.success('注册成功')
    isRegister.value = false
}

const login = async () => {
    await form.value.validate()
    ElMessage.success('登录成功')
    router.push('/admin')
}

const rules = {
    email: [
        { required: true, message: '请输入邮箱', trigger: 'change' },
        {
            pattern: /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/,
            message: '邮箱格式不正确',
            trigger: 'blur'
        },
    ],
    password: [
        { required: true, message: '请输入密码', trigger: 'change' },
        { pattern: /^\S{6,15}$/, message: '密码必须是6-15位非空字符', trigger: "blur" }
    ],
    repassword: [
        { required: true, message: '请输入密码', trigger: 'change' },
        { pattern: /^\S{6,15}$/, message: '密码必须是6-15位非空字符', trigger: "blur" },
        {
            validator: (rule, value, callback) => {
                if (value !== formModel.value.password) {
                    callback(new Error('两次输入的密码不一致'))
                } else {
                    callback()
                }
            }, trigger: "blur"
        }
    ]
}

watch(isRegister, () => {
    formModel.value = {
        email: '',
        password: '',
        repassword: ''
    }
})
</script>

<style scoped>
.login-page {
    height: 100vh;
    background-color: white;

}

.bg {
    background-image: url('@/assets/images/logo.png');
    background-repeat: no-repeat;
    background-position: 100% 100%;
}

.form {
    display: flex;
    align-items: center;
}

.form :deep(.el-form) {
    width: 100%;
}

.flex {
    width: 100%;
    display: flex;
    justify-content: space-between;
}

.button {
    width: 100%;
}
</style>