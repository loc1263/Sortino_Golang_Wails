// Punto de entrada del frontend: monta el componente raíz (App) dentro del
// <div id="app"> de index.html. En Svelte 5 se usa mount(); el antiguo
// `new App()` de Svelte 4 ya no funciona.
import { mount } from 'svelte'
import App from './App.svelte'

mount(App, { target: document.getElementById('app') })
