module.exports = {
  purge: ["./pkg/tmpl/**/*.gohtml", "./static/*.js"],
  theme: {
    extend: {},
  },
  variants: {
    height: ["responsive", "hover"],
    width: ["responsive", "hover"],
    scale: ["responsive", "hover", "focus", "group-hover"],
    display: ["responsive", "group-hover"],
  },
  plugins: [],
};
