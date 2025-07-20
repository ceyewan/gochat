<template>
    <div id="app">
        <router-view />
    </div>
</template>

<script>
export default {
    name: 'App'
}
</script>

<style>
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', 'Oxygen', 'Ubuntu', 'Cantarell', 'Fira Sans', 'Droid Sans', 'Helvetica Neue', sans-serif;
    background-color: #f5f5f5;
    color: #333;
}

#app {
    height: 100vh;
    width: 100vw;
    overflow: hidden;
}

/* 通用样式 */
.btn {
    padding: 8px 16px;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    transition: all 0.2s;
}

.btn:hover {
    opacity: 0.8;
}

.btn-primary {
    background-color: #0078ff;
    color: white;
}

.btn-secondary {
    background-color: #6c757d;
    color: white;
}
</style>
