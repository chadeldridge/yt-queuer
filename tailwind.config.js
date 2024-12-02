/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./public/*.html", "./public/**/*.html", "./public/*.js", "./public/**/*.js"],
  theme: {
    extend: {
      primary: {
        base: "#1e293b",
        dark: "#0f0f0f",
        light: "#334155",
        highlight: "#64748b",
        //highlight: "#e2e8f0",
      },
      secondary: {
        base: "#38bdf8",
        dark: "#0284c7",
        light: "#bae6fd",
        highlight: "#e0f2fe",
      },
      text: {
        light: "#f1f1f1",
        dark: "#94a3b8",
        disabled: "#64748b",
        highlight: "#ffffff",
      },
      success: "#23d18b",
      error: "#f86539",
    },
  },
  plugins: [],
}

