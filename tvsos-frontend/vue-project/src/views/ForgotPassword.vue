<template>
    <el-row class="login-page">
        <el-col :span="12" class="bg"></el-col>
        <el-col :span="6" :offset="3" class="form">
            <el-form ref="form" size="large" autocomplete="off" :model="formModel" :rules="rules">
                <el-form-item>
                    <h1>重置密码</h1>
                </el-form-item>

                <el-form-item prop="email">
                    <el-input :prefix-icon="Message" placeholder="请输入邮箱" v-model="formModel.email"></el-input>
                </el-form-item>
                <el-form-item prop="code" class="code-item">
                    <div class="code-container">
                        <el-input :prefix-icon="Check" placeholder="请输入验证码" v-model="formModel.code"
                            class="code-input"></el-input>
                        <el-button type="primary" auto-insert-space @click="sendCode" :disabled="isCodeDisabled"
                            class="send-code-btn">
                            {{ codeText }}
                        </el-button>
                    </div>
                </el-form-item>
                <el-form-item prop="password">
                    <el-input name="password" :prefix-icon="Lock" type="password" placeholder="请输入新密码"
                        v-model="formModel.password" show-password></el-input>
                </el-form-item>
                <el-form-item prop="repassword">
                    <el-input name="password" :prefix-icon="Lock" type="password" placeholder="请再次输入新密码"
                        v-model="formModel.repassword" show-password></el-input>
                </el-form-item>
                <el-form-item>
                    <el-button class="button" type="primary" auto-insert-space @click="resetPassword">重置密码</el-button>
                </el-form-item>
                <el-form-item>
                    <el-link type="primary" :underline="false" @click="goBack">← 返回登录</el-link>
                </el-form-item>
            </el-form>
        </el-col>
    </el-row>
</template>

<script setup>
import { Lock, Message, Check } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus';
import { ref, onUnmounted } from 'vue';
import router from '@/router';

const isCodeDisabled = ref(false)
const codeText = ref('发送验证码')
let countdownTimer = null
const form = ref(null)
const formModel = ref({
    email: '',
    code: '',
    password: '',
    repassword: ''

})
// 发送验证码
const sendCode = async () => {
    // 简单验证邮箱是否填写
    if (!formModel.value.email) {
        ElMessage.warning('请先输入邮箱')
        return
    }

    // 模拟发送验证码请求
    try {
        // 实际项目中替换为真实接口调用
        // await sendVerificationCode(formModel.value.email)
        ElMessage.success('验证码已发送，请注意查收')

        // 开始倒计时
        let countdown = 60
        isCodeDisabled.value = true
        codeText.value = `${countdown}s后重新发送`

        countdownTimer = setInterval(() => {
            countdown--
            codeText.value = `${countdown}s后重新发送`
            if (countdown <= 0) {
                clearInterval(countdownTimer)
                isCodeDisabled.value = false
                codeText.value = '发送验证码'
            }
        }, 1000)
    } catch (error) {
        ElMessage.error('发送失败，请重试')
        isCodeDisabled.value = false
    }
}
const resetPassword = async () => {
    try {
        await form.value.validate()
        // 实际项目中替换为真实接口调用
        // await doResetPassword(formModel.value)
        ElMessage.success('密码重置成功')
        router.push('/login') // 跳转到登录页
    } catch (error) {
        // 表单验证失败不做处理，Element Plus会自动提示错误
    }
}
const goBack = () => {
    router.push('/login')
}

//整个表单校验规则
const rules = {
    email: [
        { required: true, message: '请输入邮箱', trigger: 'change' },//非空检验
        {
            pattern: /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/,
            message: '邮箱格式不正确',
            trigger: 'blur'
        },//2.正则校验
    ],
    code: [  // 新增验证码校验规则
        { required: true, message: '请输入验证码', trigger: 'change' },
        { min: 6, max: 6, message: '验证码长度为6位', trigger: 'blur' }
    ],
    password: [
        { required: true, message: '请输入密码', trigger: 'change' },
        { pattern: /^\S{6,15}$/, message: '密码必须是6-15位非空字符', trigger: "blur" }
    ],
    repassword: [
        { required: true, message: '请输入密码', trigger: 'change' },
        { pattern: /^\S{6,15}$/, message: '密码必须是6-15位非空字符', trigger: "blur" },
        {
            validator: (rule, value, callback) => {//自定义校验 
                if (value !== formModel.value.password) {
                    callback(new Error('两次输入的密码不一致'))
                } else {
                    callback()//校验成功也需要callback
                }
            }, trigger: "blur"
        }
    ]
}
onUnmounted(() => {
    if (countdownTimer) {
        clearInterval(countdownTimer)
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

.code-container {
    display: flex;
    gap: 10px;
    /* 输入框与按钮之间的间距 */
}

.code-input {
    flex: 1;
    /* 输入框占满剩余空间 */
}

.send-code-btn {
    width: 120px;
    /* 固定按钮宽度 */
    white-space: nowrap;
    /* 防止按钮文字换行 */
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