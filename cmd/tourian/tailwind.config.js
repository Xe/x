/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./tmpl/*.html", "./tmpl/**/*.html", "./static/**/*.html", "./**/*.templ", "./*.templ"],
  theme: {
    extend: {
      fontFamily: {
        sans: ["Inter", "sans-serif"],
        mono: ["Iosevka Curly Iaso", "monospace"],
        serif: ["Podkova", "serif"],
      },
    },
  },
  plugins: [require('@tailwindcss/typography')],
};
