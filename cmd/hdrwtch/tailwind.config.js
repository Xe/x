/** @type {import('tailwindcss').Config} */
module.exports = {
    content: ["./*.templ", "./docs/**/*.md"],
    theme: {
        extend: {
            fontFamily: {
                sans: ["Iosevka Aile Iaso", "sans-serif"],
                mono: ["Iosevka Curly Iaso", "monospace"],
                serif: ["Podkova", "serif"],
            },
        },
    },
    plugins: [
        require("@tailwindcss/typography"),
        require("@tailwindcss/forms"),
    ],
};
