const colors = require("tailwindcss/colors");

/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "templates/*.templ",
    "templates/*.go",
    "static/**/*.js",
    "static/content/blogs/*.md",
  ],
  theme: {
    container: {
      center: true,
      padding: {
        DEFAULT: "1rem",
        mobile: "2rem",
        tablet: "4rem",
        desktop: "5rem",
      },
    },
    extend: {
      colors: {
        primary: colors.blue,
        secondary: colors.yellow,
        neutral: colors.gray,
      },
      typography: (theme) => ({
        DEFAULT: {
          css: {
            img: {
              maxWidth: "100%",
              height: "auto",
              marginTop: theme("spacing.4"),
              marginBottom: theme("spacing.4"),
              borderRadius: theme("borderRadius.lg"),
              display: "block",
            },
          },
        },
        invert: {
          css: {
            img: {
              maxWidth: "100%",
              height: "auto",
              marginTop: theme("spacing.4"),
              marginBottom: theme("spacing.4"),
              borderRadius: theme("borderRadius.lg"),
              display: "block",
            },
          },
        },
      }),
    },
  },
  plugins: [require("@tailwindcss/forms"), require("@tailwindcss/typography")],
};
