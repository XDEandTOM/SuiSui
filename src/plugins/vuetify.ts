import { createVuetify } from "vuetify"
import "@mdi/font/css/materialdesignicons.css"
import "vuetify/styles"

export default createVuetify({
  display: {
    mobileBreakpoint: 768
  },
  theme: {
    defaultTheme: "system",
    themes: {
      light: {
        colors: {
          background: "#F2F2F2",
          surface: "#FAFAFA",
        },
      },
      dark: {
        colors: {
          background: "#121212",
          surface: "#1E1E1E",
        },
      },
    },
  },
})