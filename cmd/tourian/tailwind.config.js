/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./tmpl/*.html", "./tmpl/**/*.html", "./static/**/*.html"],
  theme: {
    extend: {
      fontFamily: {
        sans: ["Iosevka Aile Iaso", "sans-serif"],
        mono: ["Iosevka Curly Iaso", "monospace"],
        serif: ["Podkova", "serif"],
      },
    },
  },
  plugins: [require('@tailwindcss/typography')],
};
