// @ts-check
import { defineConfig } from "astro/config"

import svelte from "@astrojs/svelte"

import tailwind from "@astrojs/tailwind"

import icon from "astro-icon"

// https://astro.build/config
export default defineConfig({
  integrations: [svelte(), tailwind(), icon()],
  // TODO I guess maybe not cos I'm gonna build static and serve
  vite: { server: { cors: { origin: "*" } } },
})
