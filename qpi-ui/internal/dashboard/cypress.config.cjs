const { defineConfig } = require("cypress");

module.exports = defineConfig({
   component: {
    devServer: {
      framework: 'react',
      bundler: 'vite',
    },
  },
  e2e: {
    baseUrl: "http://127.0.0.1:8090/dashboard/",
    supportFile: false,
    specPattern: "cypress/e2e/**/*.cy.{js,jsx,ts,tsx}",
    
  },
});
