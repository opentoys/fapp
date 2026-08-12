import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import './styles/fonts.css'
import './styles/tokens.css'
import './styles/overrides.scss'

createApp(App).use(router).mount('#app')
