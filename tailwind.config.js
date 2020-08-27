module.exports = {
  purge: {
    enabled: true,
    content: ["./pkg/tmpl/**/*.gohtml"],
  },
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
