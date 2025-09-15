import {defineStore} from 'pinia'
import {ref } from 'vue'
export const useCounterStore=defineStore('big-count',()=>{
    const count=ref(100)
    const add=(n)=>{
        count.value+=n
    }
    
    return{
        count ,add
    }
},
{
    persist:true
})//持久化