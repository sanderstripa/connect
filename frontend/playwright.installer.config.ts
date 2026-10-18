import {defineConfig} from '@playwright/test';
export default defineConfig({testDir:'./tests',testMatch:'installer.spec.ts',workers:1,use:{baseURL:'http://installer.test',viewport:{width:840,height:540},launchOptions:process.env.CONNECT_USE_CHROME?{channel:'chrome'}:{}}});
